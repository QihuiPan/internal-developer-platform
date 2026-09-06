const health = document.querySelector("#health");
const version = document.querySelector("#version");
const form = document.querySelector("#service-form");
const responseBox = document.querySelector("#response");
const operationSummary = document.querySelector("#operation-summary");
const stepItems = [...document.querySelectorAll("#steps li")];
const services = document.querySelector("#services");
const credentials = document.querySelector("#credentials");
const authDescription = document.querySelector("#auth-description");
let authMode = "demo";

function requestHeaders(contentType = false) {
  const headers = {"X-Actor": "portal-user"};
  if (authMode === "demo") headers["X-Role"] = "developer";
  const token = form.elements.token.value.trim();
  if (token) headers.Authorization = `Bearer ${token}`;
  if (contentType) headers["Content-Type"] = "application/json";
  return headers;
}

async function readJSON(response) {
  const body = await response.json();
  if (!response.ok) throw new Error(`${body.code || response.status}: ${body.message || response.statusText}`);
  return body;
}

async function checkHealth() {
  try {
    const [healthResponse, configResponse] = await Promise.all([fetch("/healthz"), fetch("/v1/config")]);
    if (!healthResponse.ok || !configResponse.ok) throw new Error("Health check failed");
    const config = await configResponse.json();
    authMode = config.authMode;
    credentials.open = authMode === "token";
    authDescription.textContent = authMode === "token" ? "Bearer token authentication is enabled." : "Local demo identity is enabled.";
    version.textContent = `Version ${config.version}`;
    health.classList.add("ready");
    health.lastChild.textContent = " API ready";
  } catch {
    health.classList.remove("ready");
    health.lastChild.textContent = " API unavailable";
  }
}

function descriptorFrom(formData) {
  const resources = [];
  if (formData.get("postgres")) resources.push({type: "postgres", plan: "small"});
  if (formData.get("redis")) resources.push({type: "redis", plan: "small"});
  return {
    apiVersion: "platform.demo/v1",
    kind: "Service",
    metadata: {name: formData.get("name"), owner: formData.get("owner")},
    spec: {
      template: formData.get("template"),
      runtime: {port: 8080, replicas: 2},
      resources,
      environments: ["dev", "staging", "production"],
      observability: {availabilitySLO: Number(formData.get("slo"))}
    }
  };
}

function renderOperation(operation) {
  responseBox.textContent = JSON.stringify(operation, null, 2);
  operationSummary.textContent = `${operation.status} - attempt ${operation.attempt}`;
  operation.steps.forEach((step, index) => {
    stepItems[index].className = step.status === "SUCCEEDED" ? "done" : step.status === "RUNNING" ? "running" : step.status === "FAILED" ? "failed" : "";
  });
}

async function pollOperation(id) {
  for (let attempt = 0; attempt < 120; attempt += 1) {
    const operation = await readJSON(await fetch(`/v1/operations/${id}`, {headers: requestHeaders()}));
    renderOperation(operation);
    if (["SUCCEEDED", "FAILED"].includes(operation.status)) {
      await loadServices();
      return operation;
    }
    await new Promise(resolve => setTimeout(resolve, 500));
  }
  throw new Error("Operation polling timed out after 60 seconds");
}

function escapeHTML(value) {
  return value.replace(/[&<>'"]/g, character => ({"&": "&amp;", "<": "&lt;", ">": "&gt;", "'": "&#39;", '"': "&quot;"})[character]);
}

async function loadServices() {
  try {
    const result = await readJSON(await fetch("/v1/services", {headers: requestHeaders()}));
    if (result.items.length === 0) {
      services.innerHTML = '<p class="empty">No services have been created.</p>';
      return;
    }
    services.innerHTML = result.items.map(item => {
      const name = escapeHTML(item.descriptor.metadata.name);
      const owner = escapeHTML(item.descriptor.metadata.owner);
      const download = item.status === "READY" ? `<button class="secondary download" type="button" data-service="${name}">Download ZIP</button>` : "";
      return `<article class="service-card"><div><h3>${name}</h3><p>${owner} · ${escapeHTML(item.descriptor.spec.template)}</p></div><div class="card-actions"><span class="badge ${item.status.toLowerCase()}">${item.status}</span>${download}</div></article>`;
    }).join("");
  } catch (error) {
    services.innerHTML = `<p class="empty error">${escapeHTML(error.message)}</p>`;
  }
}

form.addEventListener("submit", async event => {
  event.preventDefault();
  const button = form.querySelector("button[type=submit]");
  button.disabled = true;
  responseBox.textContent = "Submitting desired state...";
  stepItems.forEach(item => item.className = "");
  try {
    const descriptor = descriptorFrom(new FormData(form));
    const result = await readJSON(await fetch("/v1/services", {
      method: "POST",
      headers: {...requestHeaders(true), "Idempotency-Key": `${descriptor.metadata.name}-${crypto.randomUUID()}`},
      body: JSON.stringify(descriptor)
    }));
    renderOperation(result.operation);
    await pollOperation(result.operation.id);
  } catch (error) {
    operationSummary.textContent = "Request failed";
    responseBox.textContent = error.message;
  } finally {
    button.disabled = false;
  }
});

document.querySelector("#refresh").addEventListener("click", loadServices);
services.addEventListener("click", async event => {
  const button = event.target.closest("button.download");
  if (!button) return;
  button.disabled = true;
  try {
    const service = button.dataset.service;
    const response = await fetch(`/v1/services/${service}/archive`, {headers: requestHeaders()});
    if (!response.ok) await readJSON(response);
    const url = URL.createObjectURL(await response.blob());
    const link = document.createElement("a");
    link.href = url;
    link.download = `${service}.zip`;
    link.click();
    URL.revokeObjectURL(url);
  } catch (error) {
    responseBox.textContent = error.message;
  } finally {
    button.disabled = false;
  }
});
form.elements.token.addEventListener("change", loadServices);
checkHealth().then(loadServices);
setInterval(checkHealth, 15000);
