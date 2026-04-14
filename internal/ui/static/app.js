const API_BASE_STORAGE = "qa_api_base";

const requestLog = document.getElementById("requestLog");
const apiBaseInput = document.getElementById("apiBase");

function getApiBase() {
  const stored = localStorage.getItem(API_BASE_STORAGE);
  return (stored && stored.trim()) || window.location.origin;
}

function setApiBase(base) {
  localStorage.setItem(API_BASE_STORAGE, base.trim());
  apiBaseInput.value = base.trim();
}

function parseTags(raw) {
  return raw
    .split(",")
    .map((x) => x.trim())
    .filter(Boolean);
}

function printResult(targetId, data, ok = true) {
  const el = document.getElementById(targetId);
  el.textContent = typeof data === "string" ? data : JSON.stringify(data, null, 2);
  el.classList.remove("ok", "error");
  el.classList.add(ok ? "ok" : "error");
}

function appendLog(text) {
  const now = new Date().toISOString();
  requestLog.textContent = `[${now}] ${text}\n${requestLog.textContent}`;
}

async function apiCall(path, options = {}) {
  const method = options.method || "GET";
  const url = `${getApiBase()}${path}`;
  appendLog(`${method} ${url}`);

  const res = await fetch(url, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...(options.headers || {}),
    },
  });

  const text = await res.text();
  let body;
  try {
    body = text ? JSON.parse(text) : {};
  } catch {
    body = { raw: text };
  }

  if (!res.ok) {
    const error = new Error(body.message || `HTTP ${res.status}`);
    error.detail = body;
    throw error;
  }

  return body;
}

function formToObject(form) {
  return Object.fromEntries(new FormData(form).entries());
}

function optionalNum(v) {
  if (v === undefined || v === null || String(v).trim() === "") return undefined;
  return Number(v);
}

function optionalBool(v) {
  if (v === undefined || v === null || String(v).trim() === "") return undefined;
  return String(v) === "true";
}

function bind() {
  apiBaseInput.value = getApiBase();

  document.getElementById("saveApiBase").addEventListener("click", () => {
    setApiBase(apiBaseInput.value || window.location.origin);
    appendLog(`API base saved: ${getApiBase()}`);
  });

  document.querySelector('[data-action="health"]').addEventListener("click", async () => {
    try {
      const data = await apiCall("/health");
      printResult("healthResult", data, true);
    } catch (err) {
      printResult("healthResult", err.detail || err.message, false);
    }
  });

  document.querySelector('[data-action="ready"]').addEventListener("click", async () => {
    try {
      const data = await apiCall("/ready");
      printResult("healthResult", data, true);
    } catch (err) {
      printResult("healthResult", err.detail || err.message, false);
    }
  });

  document.getElementById("searchForm").addEventListener("submit", async (e) => {
    e.preventDefault();
    try {
      const raw = formToObject(e.target);
      const params = new URLSearchParams();
      Object.entries(raw).forEach(([k, v]) => {
        if (!String(v).trim()) return;
        if (k === "tags") {
          parseTags(v).forEach((tag) => params.append("tags", tag));
          return;
        }
        params.append(k, v);
      });
      const data = await apiCall(`/v1/products?${params.toString()}`);
      printResult("searchResult", data, true);
    } catch (err) {
      printResult("searchResult", err.detail || err.message, false);
    }
  });

  document.getElementById("createForm").addEventListener("submit", async (e) => {
    e.preventDefault();
    try {
      const raw = formToObject(e.target);
      const payload = {
        name: raw.name,
        description: raw.description || "",
        category: raw.category,
        brand: raw.brand || "",
        price: Number(raw.price),
        stock: optionalNum(raw.stock) || 0,
        tags: parseTags(raw.tags || ""),
        is_active: e.target.is_active.checked,
      };
      const data = await apiCall("/v1/products", {
        method: "POST",
        body: JSON.stringify(payload),
      });
      printResult("createResult", data, true);
      if (data?.id) {
        appendLog(`Created product id=${data.id}`);
      }
    } catch (err) {
      printResult("createResult", err.detail || err.message, false);
    }
  });

  document.getElementById("getForm").addEventListener("submit", async (e) => {
    e.preventDefault();
    try {
      const raw = formToObject(e.target);
      const data = await apiCall(`/v1/products/${encodeURIComponent(raw.id)}`);
      printResult("getResult", data, true);
    } catch (err) {
      printResult("getResult", err.detail || err.message, false);
    }
  });

  document.getElementById("updateForm").addEventListener("submit", async (e) => {
    e.preventDefault();
    try {
      const raw = formToObject(e.target);
      const payload = {
        name: raw.name || undefined,
        description: raw.description || undefined,
        category: raw.category || undefined,
        brand: raw.brand || undefined,
        price: optionalNum(raw.price),
        stock: optionalNum(raw.stock),
        tags: raw.tags ? parseTags(raw.tags) : undefined,
        is_active: optionalBool(raw.is_active),
      };
      Object.keys(payload).forEach((k) => payload[k] === undefined && delete payload[k]);

      const data = await apiCall(`/v1/products/${encodeURIComponent(raw.id)}`, {
        method: "PUT",
        body: JSON.stringify(payload),
      });
      printResult("updateResult", data, true);
    } catch (err) {
      printResult("updateResult", err.detail || err.message, false);
    }
  });

  document.getElementById("deleteForm").addEventListener("submit", async (e) => {
    e.preventDefault();
    try {
      const raw = formToObject(e.target);
      await apiCall(`/v1/products/${encodeURIComponent(raw.id)}`, {
        method: "DELETE",
      });
      printResult("deleteResult", { message: "deleted (204)" }, true);
    } catch (err) {
      printResult("deleteResult", err.detail || err.message, false);
    }
  });

  document.getElementById("bulkForm").addEventListener("submit", async (e) => {
    e.preventDefault();
    try {
      const payload = JSON.parse(document.getElementById("bulkPayload").value);
      const data = await apiCall("/v1/products/bulk", {
        method: "POST",
        body: JSON.stringify(payload),
      });
      printResult("bulkResult", data, true);
    } catch (err) {
      printResult("bulkResult", err.detail || err.message, false);
    }
  });
}

bind();
