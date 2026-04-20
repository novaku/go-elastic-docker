const API_BASE_STORAGE = "qa_api_base";
const JWT_TOKEN_STORAGE = "qa_jwt_token";

const requestLog = document.getElementById("requestLog");
const apiBaseInput = document.getElementById("apiBase");
const loginOverlay = document.getElementById("loginOverlay");
const authStatus = document.getElementById("authStatus");
const logoutBtn = document.getElementById("logoutBtn");

function getApiBase() {
  const stored = localStorage.getItem(API_BASE_STORAGE);
  return (stored && stored.trim()) || window.location.origin;
}

function setApiBase(base) {
  localStorage.setItem(API_BASE_STORAGE, base.trim());
  apiBaseInput.value = base.trim();
}

function getToken() {
  return localStorage.getItem(JWT_TOKEN_STORAGE) || "";
}

function setToken(token) {
  localStorage.setItem(JWT_TOKEN_STORAGE, token);
}

function clearToken() {
  localStorage.removeItem(JWT_TOKEN_STORAGE);
}

function isTokenValid() {
  const token = getToken();
  if (!token) return false;
  try {
    const payload = JSON.parse(atob(token.split(".")[1]));
    return payload.exp * 1000 > Date.now();
  } catch {
    return false;
  }
}

function updateAuthUI() {
  if (isTokenValid()) {
    try {
      const payload = JSON.parse(atob(getToken().split(".")[1]));
      authStatus.textContent = `Logged in as: ${payload.username}`;
    } catch {
      authStatus.textContent = "Logged in";
    }
    authStatus.className = "auth-status authenticated";
    logoutBtn.style.display = "";
    loginOverlay.style.display = "none";
  } else {
    clearToken();
    authStatus.textContent = "Not logged in";
    authStatus.className = "auth-status unauthenticated";
    logoutBtn.style.display = "none";
    loginOverlay.style.display = "flex";
  }
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

  const headers = {
    "Content-Type": "application/json",
    ...(options.headers || {}),
  };

  const token = getToken();
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const res = await fetch(url, { ...options, headers });

  if (res.status === 401) {
    clearToken();
    updateAuthUI();
    const error = new Error("Unauthorized — please log in again");
    error.detail = { message: "Session expired or invalid token. Please log in." };
    throw error;
  }

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
  updateAuthUI();

  document.getElementById("loginForm").addEventListener("submit", async (e) => {
    e.preventDefault();
    const loginError = document.getElementById("loginError");
    loginError.style.display = "none";
    try {
      const data = await fetch(`${getApiBase()}/auth/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          username: document.getElementById("loginUsername").value,
          password: document.getElementById("loginPassword").value,
        }),
      });
      const body = await data.json();
      if (!data.ok) {
        loginError.textContent = body.message || `HTTP ${data.status}`;
        loginError.style.display = "";
        loginError.classList.add("error");
        return;
      }
      setToken(body.token);
      document.getElementById("loginPassword").value = "";
      appendLog(`Logged in, token expires in ${body.expires_in}s`);
      updateAuthUI();
    } catch (err) {
      loginError.textContent = err.message;
      loginError.style.display = "";
      loginError.classList.add("error");
    }
  });

  logoutBtn.addEventListener("click", () => {
    clearToken();
    appendLog("Logged out");
    updateAuthUI();
  });

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

  // ── Live search (keyup) ────────────────────────────────────────────────
  const liveQ = document.getElementById("liveQ");
  const liveResults = document.getElementById("liveResults");
  const liveCount = document.getElementById("liveCount");
  const liveSpinner = document.getElementById("liveSpinner");
  let liveTimer = null;

  function renderProducts(hits) {
    if (!hits || hits.length === 0) {
      liveResults.innerHTML = '<p class="no-hits">No products found.</p>';
      return;
    }
    liveResults.innerHTML = hits
      .map(
        (p) => `
        <div class="product-card">
          <div class="product-name">${escHtml(p.name || "")}</div>
          <div class="product-meta">
            <span class="tag cat">${escHtml(p.category || "")}</span>
            ${p.brand ? `<span class="tag brand">${escHtml(p.brand)}</span>` : ""}
            <span class="tag price">$${Number(p.price || 0).toFixed(2)}</span>
            ${(p.tags || []).map((t) => `<span class="tag">${escHtml(t)}</span>`).join("")}
          </div>
          <div class="product-desc">${escHtml(p.description || "")}</div>
          <div class="product-id">ID: ${escHtml(p.id || "")}</div>
        </div>`
      )
      .join("");
  }

  function escHtml(str) {
    return String(str).replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));
  }

  async function doLiveSearch(q) {
    if (!q.trim()) {
      liveResults.innerHTML = "";
      liveCount.textContent = "";
      return;
    }
    liveSpinner.style.display = "";
    try {
      const params = new URLSearchParams({ q, page_size: 20 });
      const data = await apiCall(`/v1/products?${params}`);
      const hits = data.products || data.data || data.hits || data || [];
      const total = data.total ?? (Array.isArray(hits) ? hits.length : 0);
      liveCount.textContent = `${total} result${total !== 1 ? "s" : ""} for "${q}"`;
      renderProducts(Array.isArray(hits) ? hits : []);
      appendLog(`Live search: q="${q}" → ${total} hits`);
    } catch (err) {
      liveResults.innerHTML = `<p class="no-hits error">${escHtml(err.message)}</p>`;
      liveCount.textContent = "";
    } finally {
      liveSpinner.style.display = "none";
    }
  }

  liveQ.addEventListener("keyup", () => {
    clearTimeout(liveTimer);
    liveTimer = setTimeout(() => doLiveSearch(liveQ.value), 300);
  });
  // ────────────────────────────────────────────────────────────────────────

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
