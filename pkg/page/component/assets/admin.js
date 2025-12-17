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
          trigger_type: "manual",
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
    const name = (f.querySelector("input[name=name]").value || "").trim();
    const description = (
      f.querySelector("input[name=description]").value || ""
    ).trim();
    const enabled = !!f.querySelector("input[name=enabled]").checked;

    const triggerType = f.querySelector("select[name=trigger_type]").value;
    const webhookToken = (
      f.querySelector("input[name=webhook_token]").value || ""
    ).trim();
    const cronSpec = (
      f.querySelector("input[name=cron_spec]").value || ""
    ).trim();

    const action1El =
      f.querySelector("select[name=action1]") ||
      f.querySelector("input[name=action1]");
    const action1 = action1El ? action1El.value : "";
    const action1Params = readJsonFromTextarea("action1_params");
    const action2El =
      f.querySelector("select[name=action2]") ||
      f.querySelector("input[name=action2]");
    const action2 = action2El ? action2El.value : "";
    const action2Params = readJsonFromTextarea("action2_params");

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
    const triggerParams = {};
    if (triggerType === "webhook") triggerParams.token = webhookToken;
    if (triggerType === "cron") triggerParams.spec = cronSpec;

    nodes.push({
      node_id: triggerNodeId,
      type: "trigger",
      bot: "system",
      rule_id: triggerType,
      label: triggerType,
      parameters: triggerParams,
    });

    let prev = triggerNodeId;
    function addAction(nodeId, actionValue, params) {
      if (!actionValue) return;
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

    addAction("action1", action1, action1Params);
    addAction("action2", action2, action2Params);

    await jsonFetch(serviceUrl("/service/flows/" + id + "/nodes"), {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ nodes: nodes, edges: edges }),
    });

    notify("Flow saved", "success");
    location.href =
      "/page/flows_list/" + (f.querySelector("input[name=flag]").value || "");
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
