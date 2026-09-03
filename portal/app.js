const apiBase = location.port === "3000" ? "/api" : "http://localhost:8080";
const health = document.querySelector("#health");
const form = document.querySelector("#service-form");
const responseBox = document.querySelector("#response");
const operationSummary = document.querySelector("#operation-summary");
const stepItems = [...document.querySelectorAll("#steps li")];

const headers = { "Content-Type": "application/json", "X-Actor": "portal-user", "X-Role": "developer" };

async function checkHealth() {
  try {
    const response = await fetch(`${apiBase}/healthz`);
    if (!response.ok) throw new Error("Health check failed");
    health.classList.add("ready");
    health.lastChild.textContent = " API ready";
  } catch {
    health.classList.remove("ready");
    health.lastChild.textContent = " API unavailable";
  }
}

function descriptorFrom(formData) {
  const resources = [];
  if (formData.get("postgres")) resources.push({ type: "postgres", plan: "small" });
  if (formData.get("redis")) resources.push({ type: "redis", plan: "small" });
  return {
    apiVersion: "platform.demo/v1",
    kind: "Service",
    metadata: { name: formData.get("name"), owner: formData.get("owner") },
    spec: {
      template: formData.get("template"),
      runtime: { port: 8080, replicas: 2 },
      resources,
      environments: ["dev", "staging", "production"],
      observability: { availabilitySLO: Number(formData.get("slo")) }
    }
  };
}

function renderOperation(operation) {
  responseBox.textContent = JSON.stringify(operation, null, 2);
  operationSummary.textContent = `${operation.status} - attempt ${operation.attempt}`;
  operation.steps.forEach((step, index) => {
    stepItems[index].className = step.status === "SUCCEEDED" ? "done" : step.status === "RUNNING" ? "running" : "";
  });
}

async function pollOperation(id) {
  for (let attempt = 0; attempt < 60; attempt += 1) {
    const response = await fetch(`${apiBase}/v1/operations/${id}`, { headers });
    const operation = await response.json();
    renderOperation(operation);
    if (["SUCCEEDED", "FAILED"].includes(operation.status)) return operation;
    await new Promise(resolve => setTimeout(resolve, 500));
  }
  throw new Error("Operation polling timed out");
}

form.addEventListener("submit", async event => {
  event.preventDefault();
  const button = form.querySelector("button");
  button.disabled = true;
  responseBox.textContent = "Submitting desired state...";
  stepItems.forEach(item => item.className = "");
  try {
    const descriptor = descriptorFrom(new FormData(form));
    const response = await fetch(`${apiBase}/v1/services`, {
      method: "POST",
      headers: { ...headers, "Idempotency-Key": `${descriptor.metadata.name}-${crypto.randomUUID()}` },
      body: JSON.stringify(descriptor)
    });
    const result = await response.json();
    if (!response.ok) throw new Error(`${result.code}: ${result.message}`);
    renderOperation(result.operation);
    await pollOperation(result.operation.id);
  } catch (error) {
    operationSummary.textContent = "Request failed";
    responseBox.textContent = error.message;
  } finally {
    button.disabled = false;
  }
});

checkHealth();
setInterval(checkHealth, 15000);
