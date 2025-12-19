function notify(message, status) {
  if (typeof UIkit !== "undefined" && UIkit.notification) {
    UIkit.notification(message, { status: status || "primary" });
    return;
  }
  alert(message);
}

function currentFlag() {
  // pages are rendered under /page/:id/:flag
  const parts = (window.location.pathname || "").split("/").filter(Boolean);
  if (parts.length >= 3 && parts[0] === "page") {
    return parts[2];
  }
  return "";
}

function serviceUrl(path) {
  const p = currentFlag();
  if (!p) return path;
  return (
    path + (path.indexOf("?") >= 0 ? "&" : "?") + "p=" + encodeURIComponent(p)
  );
}

async function jsonFetch(url, options) {
  const res = await fetch(url, options || {});
  const text = await res.text();
  let data = null;
  try {
    data = text ? JSON.parse(text) : null;
  } catch (e) {
    data = { raw: text };
  }
  if (!res.ok) {
    const msg = (data && (data.error || data.message)) || res.statusText;
    throw new Error(msg);
  }
  return data;
}

async function executeFlow(id) {
  try {
    const res = await jsonFetch(
      serviceUrl("/service/flows/" + id + "/execute"),
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          trigger_type: "dev|manual",
          trigger_id: "",
          payload: {},
        }),
      },
    );
    notify((res && res.message) || "Flow execution started", "success");
  } catch (e) {
    notify((e && e.message) || String(e), "danger");
  }
}

async function deleteFlow(id) {
  if (!confirm("Are you sure?")) return;
  await fetch(serviceUrl("/service/flows/" + id), { method: "DELETE" });
  notify("Flow deleted", "success");
  location.reload();
}

async function scanApps() {
  await jsonFetch(serviceUrl("/service/apps/scan"), { method: "POST" });
  notify("Scan completed", "success");
  location.reload();
}

async function appAction(id, action) {
  await jsonFetch(serviceUrl("/service/apps/" + id + "/" + action), {
    method: "POST",
  });
  notify("Action completed", "success");
  location.reload();
}

async function deleteConnection(id) {
  if (!confirm("Are you sure?")) return;
  await fetch(serviceUrl("/service/connections/" + id), { method: "DELETE" });
  notify("Connection deleted", "success");
  location.reload();
}

async function deleteAuth(id) {
  if (!confirm("Are you sure?")) return;
  await fetch(serviceUrl("/service/authentications/" + id), {
    method: "DELETE",
  });
  notify("Authentication deleted", "success");
  location.reload();
}

function readJsonFromTextarea(id) {
  const el = document.getElementById(id);
  if (!el) return {};
  const txt = (el.value || "").trim();
  if (!txt) return {};

  let parsed;
  try {
    parsed = JSON.parse(txt);
  } catch (e) {
    notify("Invalid JSON in " + id, "danger");
    throw e;
  }

  // The backend expects `parameters` to be a JSON object (map). If user
  // provides a primitive/array, wrap it so the payload always binds.
  if (parsed === null) return {};
  if (typeof parsed !== "object" || Array.isArray(parsed)) {
    return { value: parsed };
  }
  return parsed;
}

async function saveFlow(formId) {
  try {
    const f = document.getElementById(formId);
    const flowId = (f.querySelector("input[name=flow_id]").value || "").trim();
    const isNewFlow = !flowId;
    const name = (f.querySelector("input[name=name]").value || "").trim();
    const description = (
      f.querySelector("input[name=description]").value || ""
    ).trim();
    const enabled = !!f.querySelector("input[name=enabled]").checked;

    const trigger = (f.querySelector("select[name=trigger]") || {}).value || "";
    const triggerParams = readJsonFromTextarea("trigger_params");

    const actionEl =
      f.querySelector("select[name=action]") ||
      f.querySelector("input[name=action]");
    const action = actionEl ? actionEl.value : "";
    const actionParams = readJsonFromTextarea("action_params");

    if (!name) {
      notify("Name is required", "danger");
      return;
    }

    const flowPayload = {
      name: name,
      description: description,
      enabled: enabled,
      state: enabled ? 1 : 0,
    };

    let id = flowId;
    if (!id) {
      const created = await jsonFetch(serviceUrl("/service/flows/"), {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(flowPayload),
      });
      id = String(created.id);
    } else {
      await jsonFetch(serviceUrl("/service/flows/" + id), {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(flowPayload),
      });
    }

    const nodes = [];
    const edges = [];

    const triggerNodeId = "trigger";
    let triggerBot = "dev";
    let triggerRule = "manual";
    if (trigger && trigger.indexOf("|") >= 0) {
      const parts = trigger.split("|");
      triggerBot = parts[0];
      triggerRule = parts[1];
    } else if (trigger) {
      triggerRule = trigger;
    }

    nodes.push({
      node_id: triggerNodeId,
      type: "trigger",
      bot: triggerBot,
      rule_id: triggerRule,
      label: trigger,
      parameters: triggerParams || {},
    });

    let prev = triggerNodeId;
    function addAction(nodeId, actionValue, params) {
      if (!actionValue) return;
      if (String(actionValue).trim() === "(none)") return;
      if (String(actionValue).trim() === "") return;
      if (actionValue.indexOf("|") === -1) {
        notify("Action must be in format bot|rule_id", "danger");
        throw new Error("invalid action format");
      }
      const parts = actionValue.split("|");
      const bot = parts[0];
      const rule = parts[1];
      nodes.push({
        node_id: nodeId,
        type: "action",
        bot: bot,
        rule_id: rule,
        label: actionValue,
        parameters: params || {},
      });
      edges.push({
        edge_id: prev + "->" + nodeId,
        source_node: prev,
        target_node: nodeId,
      });
      prev = nodeId;
    }

    addAction("action", action, actionParams);

    await jsonFetch(serviceUrl("/service/flows/" + id + "/nodes"), {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ nodes: nodes, edges: edges }),
    });

    notify("Flow saved", "success");
    const flag = (f.querySelector("input[name=flag]").value || "").trim();
    if (isNewFlow) {
      // New flow: open edit page so server-generated values like webhook token are visible.
      location.assign(
        "/page/flows_edit/" + flag + "?flow_id=" + encodeURIComponent(id),
      );
      return;
    }

    // Existing flow: return to list page.
    location.assign("/page/flows_list/" + flag);
  } catch (e) {
    notify((e && e.message) || String(e), "danger");
  }
}

async function saveConnection(formId) {
  const f = document.getElementById(formId);
  const id = (f.querySelector("input[name=id]").value || "").trim();
  const name = (f.querySelector("input[name=name]").value || "").trim();
  const type = (f.querySelector("input[name=type]").value || "").trim();
  const enabled = !!f.querySelector("input[name=enabled]").checked;
  const config = readJsonFromTextarea("conn_config");

  if (!name || !type) {
    notify("Name and Type are required", "danger");
    return;
  }

  const payload = { name: name, type: type, enabled: enabled, config: config };
  if (!id) {
    await jsonFetch(serviceUrl("/service/connections/"), {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
  } else {
    await jsonFetch(serviceUrl("/service/connections/" + id), {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
  }
  notify("Connection saved", "success");
  location.href =
    "/page/connections/" + (f.querySelector("input[name=flag]").value || "");
}

function getFlowRuleMeta() {
  const el = document.getElementById("flow_rule_meta");
  if (!el) return null;
  const txt = (el.textContent || "").trim();
  if (!txt) return null;
  try {
    return JSON.parse(txt);
  } catch (e) {
    // Don't break the whole admin bundle.
    return null;
  }
}

function setReadonlyJsonText(id, obj) {
  const el = document.getElementById(id);
  if (!el) return;
  try {
    const txt = JSON.stringify(obj || {}, null, 2);
    if (typeof el.value === "string") {
      el.value = txt;
    } else {
      el.textContent = txt;
    }
  } catch (e) {
    if (typeof el.value === "string") {
      el.value = "{}";
    } else {
      el.textContent = "{}";
    }
  }
}

function safeJsonParse(txt) {
  try {
    return JSON.parse(txt);
  } catch (e) {
    return null;
  }
}

function extractIngredientNamesFromParams(params) {
  if (!params || typeof params !== "object") return [];
  const arr = params.ingredients;
  if (!Array.isArray(arr)) return [];
  const names = [];
  for (const it of arr) {
    if (!it || typeof it !== "object") continue;
    const n = String(it.name || "").trim();
    if (n) names.push(n);
  }
  // de-dupe, preserve order
  return [...new Set(names)];
}

function initFlowEditorMetaUI() {
  const meta = getFlowRuleMeta();
  if (!meta) return;

  const formEl = document.getElementById("flow_edit_form");
  // Prevent re-binding events during SPA DOM swaps.
  if (formEl && formEl.dataset && formEl.dataset.flowMetaInit === "1") {
    return;
  }

  const triggerSel = document.querySelector("select[name=trigger]");
  const actionSel = document.querySelector("select[name=action]");
  const triggerParamsEl = document.getElementById("trigger_params");

  function updateTrigger() {
    if (!triggerSel) return;
    const k = triggerSel.value || "";
    const t = (meta.triggers || {})[k] || {};

    setReadonlyJsonText("trigger_params_example", t.params_example || {});

    // Ingredients variables: prefer current Trigger Params, fallback to rule-defined ingredients.
    let ingredientNames = [];
    if (triggerParamsEl) {
      const parsed = safeJsonParse((triggerParamsEl.value || "").trim());
      ingredientNames = extractIngredientNamesFromParams(parsed);
    }
    // If user hasn't configured trigger params yet, fall back to the rule's example.
    if (ingredientNames.length === 0) {
      ingredientNames = extractIngredientNamesFromParams(t.params_example);
    }
    // Finally, fall back to any statically declared ingredients on the rule.
    if (ingredientNames.length === 0 && Array.isArray(t.ingredients)) {
      ingredientNames = [
        ...new Set(
          t.ingredients
            .map((x) => String((x || {}).name || "").trim())
            .filter(Boolean),
        ),
      ];
    }

    const varsEl = document.getElementById("trigger_ingredient_vars");
    if (varsEl) {
      if (!ingredientNames.length) {
        varsEl.textContent = "";
      } else {
        // Clear any server-rendered placeholder nodes.
        varsEl.innerHTML = "";
        varsEl.textContent = ingredientNames.map((n) => `{{${n}}}`).join(", ");
      }
    }
  }

  function updateAction() {
    if (!actionSel) return;
    const k = actionSel.value || "";
    const a = (meta.actions || {})[k] || {};
    setReadonlyJsonText("action_params_example", a.params_example || {});
  }

  if (triggerSel) triggerSel.addEventListener("change", updateTrigger);
  if (actionSel) actionSel.addEventListener("change", updateAction);
  if (triggerParamsEl) triggerParamsEl.addEventListener("input", updateTrigger);

  // Initial render.
  updateTrigger();
  updateAction();

  if (formEl && formEl.dataset) {
    formEl.dataset.flowMetaInit = "1";
  }
}

function tryInitFlowEditorMetaUI() {
  // In go-app SPA navigation, scripts persist while the DOM swaps.
  // Only initialize when the Flow Editor DOM is present.
  const hasMeta = !!document.getElementById("flow_rule_meta");
  const hasForm = !!document.getElementById("flow_edit_form");
  if (!hasMeta || !hasForm) return false;
  initFlowEditorMetaUI();
  return true;
}

// Initial attempt for first page load.
window.addEventListener("load", tryInitFlowEditorMetaUI);
tryInitFlowEditorMetaUI();

// Observe DOM changes so editor works after SPA navigation.
(() => {
  let lastInitAt = 0;
  const obs = new MutationObserver(() => {
    const now = Date.now();
    // Basic throttle to avoid excessive work during large DOM updates.
    if (now - lastInitAt < 200) return;
    if (tryInitFlowEditorMetaUI()) {
      lastInitAt = now;
    }
  });
  if (document.documentElement) {
    obs.observe(document.documentElement, { childList: true, subtree: true });
  }
})();

async function saveAuthentication(formId) {
  const f = document.getElementById(formId);
  const id = (f.querySelector("input[name=id]").value || "").trim();
  const name = (f.querySelector("input[name=name]").value || "").trim();
  const type = (f.querySelector("input[name=type]").value || "").trim();
  const enabled = !!f.querySelector("input[name=enabled]").checked;
  const credentials = readJsonFromTextarea("auth_credentials");

  if (!name || !type) {
    notify("Name and Type are required", "danger");
    return;
  }

  const payload = {
    name: name,
    type: type,
    enabled: enabled,
    credentials: credentials,
  };
  if (!id) {
    await jsonFetch(serviceUrl("/service/authentications/"), {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
  } else {
    await jsonFetch(serviceUrl("/service/authentications/" + id), {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
  }
  notify("Authentication saved", "success");
  location.href =
    "/page/authentications/" +
    (f.querySelector("input[name=flag]").value || "");
}
