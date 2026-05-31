(function () {
  function register() {
    Alpine.data("pipelineEditor", () => ({
      name: "",
      description: "",
      status: "draft",
      version: 1,
      dirty: false,
      undoStack: [],
      redoStack: [],
      triggers: [],
      steps: [],
      selectedNode: null,
      drawerOpen: false,
      drawerExpanded: false,
      drawerTab: "setup",
      drawerDirty: false,
      codeView: false,
      yamlText: "",
      variablePickerOpen: false,
      variablePickerTarget: null,
      variablePickerSource: "event",
      errors: [],
      publishDisabled: false,
      autoSaveTimer: null,
      testTriggerSource: "event",
      testMockPayload: "{}",
      testResults: null,
      capabilities: [],

      init() {
        const el = this.$el;
        const name = el.dataset.pipelineName || "";
        this.name = name;
        if (name) this.loadPipeline(name);
        this.fetchCapabilities();
        this.pushUndo();
      },

      async loadPipeline(name) {
        try {
          const resp = await fetch(`/service/web/pipelines/${name}/yaml`);
          const data = await resp.json();
          this.version = data.version;
          this.status = data.status;
          if (data.yaml) this.parseYamlToState(data.yaml);
        } catch (e) {
          console.error("Failed to load pipeline:", e);
        }
      },

      async fetchCapabilities() {
        try {
          const resp = await fetch("/service/web/pipelines/capabilities");
          const json = await resp.json();
          this.capabilities = json.data || [];
        } catch (e) {
          console.error("Failed to load capabilities:", e);
        }
      },

      getOperationsFor(capType) {
        const cap = this.capabilities.find((c) => c.type === capType);
        return cap ? cap.operations || [] : [];
      },

      getEventsForTrigger() {
        const groups = [];
        for (const cap of this.capabilities) {
          if (cap.events && cap.events.length > 0) {
            groups.push({ capability: cap.type, description: cap.description, events: cap.events });
          }
        }
        return groups;
      },

      isEventKnown(eventName) {
        if (!eventName) return true;
        for (const cap of this.capabilities) {
          if ((cap.events || []).some((e) => e.name === eventName)) return true;
        }
        return false;
      },

      parseYamlToState(yaml) {
        try {
          const obj = jsyaml.load(yaml);
          this.name = obj.name || this.name;
          this.description = obj.description || "";
          this.triggers = (obj.triggers || []).map((t) => ({
            type: t.type || "event",
            enabled: t.enabled !== false,
            event: t.event || "",
            cron: t.cron || "",
            webhook: t.webhook || {
              path: "",
              method: "POST",
              auth: { token: "", hmac_secret: "" },
            },
          }));
          this.steps = (obj.steps || []).map((s) => ({
            name: s.name || "",
            capability: s.capability || "",
            operation: s.operation || "",
            paramsText: JSON.stringify(s.params || {}, null, 2),
          }));
          this.validate();
        } catch (e) {
          console.error("YAML parse error:", e);
        }
      },

      stateToYaml() {
        const obj = {
          name: this.name,
          description: this.description,
          enabled: true,
          resumable: false,
          triggers: this.triggers.map((t) => {
            const e = { type: t.type, enabled: t.enabled };
            if (t.type === "event") e.event = t.event;
            if (t.type === "cron") e.cron = t.cron;
            if (t.type === "webhook") e.webhook = t.webhook;
            return e;
          }),
          steps: this.steps.map((s) => ({
            name: s.name,
            capability: s.capability,
            operation: s.operation,
            params: (() => {
              try {
                return JSON.parse(s.paramsText || "{}");
              } catch (e) {
                return {};
              }
            })(),
          })),
        };
        return jsyaml.dump(obj);
      },

      pushUndo() {
        this.undoStack.push(
          JSON.parse(
            JSON.stringify({ triggers: this.triggers, steps: this.steps }),
          ),
        );
        if (this.undoStack.length > 50) this.undoStack.shift();
        this.redoStack = [];
      },

      undo() {
        if (this.undoStack.length <= 1) return;
        this.redoStack.push(this.undoStack.pop());
        const prev = this.undoStack[this.undoStack.length - 1];
        this.triggers = JSON.parse(JSON.stringify(prev.triggers));
        this.steps = JSON.parse(JSON.stringify(prev.steps));
        this.markDirty();
        this.validate();
      },

      redo() {
        if (this.redoStack.length === 0) return;
        const next = this.redoStack.pop();
        this.undoStack.push(JSON.parse(JSON.stringify(next)));
        this.triggers = JSON.parse(JSON.stringify(next.triggers));
        this.steps = JSON.parse(JSON.stringify(next.steps));
        this.markDirty();
        this.validate();
      },

      addTrigger() {
        this.pushUndo();
        this.triggers.push({
          type: "event",
          enabled: true,
          event: "",
          cron: "",
          webhook: {
            path: "",
            method: "POST",
            auth: { token: "", hmac_secret: "" },
          },
        });
        this.markDirty();
      },

      removeTrigger(idx) {
        this.pushUndo();
        this.triggers.splice(idx, 1);
        this.markDirty();
        this.validate();
      },

      addStep(afterIdx) {
        this.pushUndo();
        this.steps.splice(afterIdx, 0, {
          name: "",
          capability: "",
          operation: "",
          paramsText: "{}",
        });
        this.markDirty();
        this.selectNode("step", afterIdx);
      },

      removeStep(idx) {
        this.pushUndo();
        this.steps.splice(idx, 1);
        this.markDirty();
        this.validate();
        if (this.drawerOpen && this.selectedNode?.index === idx)
          this.drawerOpen = false;
      },

      moveStepUp(idx) {
        if (idx === 0) return;
        if (this.dependsOnStep(this.steps[idx], idx - 1)) {
          alert(
            "Cannot move: this step depends on data from a step above the target position.",
          );
          return;
        }
        this.pushUndo();
        const step = this.steps.splice(idx, 1)[0];
        this.steps.splice(idx - 1, 0, step);
        this.markDirty();
        this.validate();
      },

      moveStepDown(idx) {
        if (idx >= this.steps.length - 1) return;
        if (this.dependsOnStep(this.steps[idx + 1], idx, this.steps[idx])) {
          alert("Cannot move: the step below depends on this step's data.");
          return;
        }
        this.pushUndo();
        const step = this.steps.splice(idx, 1)[0];
        this.steps.splice(idx + 1, 0, step);
        this.markDirty();
        this.validate();
      },

      duplicateStep(idx) {
        this.pushUndo();
        const copy = JSON.parse(JSON.stringify(this.steps[idx]));
        copy.name = copy.name + "-copy";
        this.steps.splice(idx + 1, 0, copy);
        this.markDirty();
      },

      dependsOnStep(step, targetIdx) {
        const re = /\{\{steps\.(\w+)\./g;
        const refs = [...(step.paramsText || "").matchAll(re)].map((m) => m[1]);
        return refs.some(
          (ref) => this.steps.findIndex((s) => s.name === ref) >= targetIdx,
        );
      },

      selectNode(type, idx) {
        if (this.drawerDirty && this.selectedNode) {
          if (!confirm("You have unsaved changes. Discard them?")) return;
        }
        this.selectedNode = { type, index: idx };
        this.drawerOpen = true;
        this.drawerDirty = false;
        this.drawerTab = "setup";
      },

      closeDrawer() {
        if (this.drawerDirty) {
          if (!confirm("You have unsaved changes. Discard them?")) return;
        }
        this.drawerOpen = false;
        this.selectedNode = null;
        this.drawerDirty = false;
      },

      toggleDrawerExpand() {
        this.drawerExpanded = !this.drawerExpanded;
      },

      openVariablePicker(targetIdx) {
        this.variablePickerTarget = targetIdx;
        this.variablePickerOpen = true;
      },

      insertVariable(path) {
        if (this.variablePickerTarget === null) return;
        const step = this.steps[this.variablePickerTarget];
        const template = "{{" + path + "}}";
        const textarea = document.querySelector(
          '[data-testid="params-editor"]',
        );
        if (textarea) {
          const start = textarea.selectionStart;
          const end = textarea.selectionEnd;
          step.paramsText =
            (step.paramsText || "").substring(0, start) +
            template +
            (step.paramsText || "").substring(end);
          setTimeout(() => {
            textarea.focus();
            textarea.setSelectionRange(
              start + template.length,
              start + template.length,
            );
          }, 50);
        } else {
          step.paramsText = (step.paramsText || "") + template;
        }
        this.variablePickerOpen = false;
        this.markDirty();
      },

      validate() {
        this.errors = [];
        if (this.triggers.filter((t) => t.enabled).length === 0)
          this.errors.push({
            node: { type: "trigger", index: -1 },
            message: "At least one trigger must be enabled",
          });
        if (this.steps.length === 0)
          this.errors.push({
            node: { type: "step", index: -1 },
            message: "At least one step is required",
          });
        for (let i = 0; i < this.triggers.length; i++) {
          const t = this.triggers[i];
          if (!t.enabled) continue;
          if (t.type === "event" && !t.event)
            this.errors.push({
              node: { type: "trigger", index: i },
              message: "Event type is required",
            });
          if (t.type === "webhook" && (!t.webhook || !t.webhook.path))
            this.errors.push({
              node: { type: "trigger", index: i },
              message: "Webhook path is required",
            });
          if (
            t.type === "webhook" &&
            t.webhook &&
            !t.webhook.auth.token &&
            !t.webhook.auth.hmac_secret
          )
            this.errors.push({
              node: { type: "trigger", index: i },
              message: "At least one auth method is required",
            });
          if (t.type === "cron" && !t.cron)
            this.errors.push({
              node: { type: "trigger", index: i },
              message: "Cron expression is required",
            });
        }
        for (let i = 0; i < this.steps.length; i++) {
          const s = this.steps[i];
          if (!s.name)
            this.errors.push({
              node: { type: "step", index: i },
              message: "Step name is required",
            });
          if (!s.capability)
            this.errors.push({
              node: { type: "step", index: i },
              message: "Capability is required",
            });
          if (!s.operation)
            this.errors.push({
              node: { type: "step", index: i },
              message: "Operation is required",
            });
          const re = /\{\{steps\.(\w+)\./g;
          const refs = [...(s.paramsText || "").matchAll(re)].map((m) => m[1]);
          for (const ref of refs) {
            const ri = this.steps.findIndex((ss) => ss.name === ref);
            if (ri === -1)
              this.errors.push({
                node: { type: "step", index: i },
                message:
                  "Upstream variable {{steps." +
                  ref +
                  ".*}} is invalid or has been removed",
              });
            else if (ri >= i)
              this.errors.push({
                node: { type: "step", index: i },
                message:
                  "Depends on [" + ref + "] which must be above this step",
              });
          }
        }
        this.publishDisabled = this.errors.length > 0;
      },

      getTriggerErrorClass(idx) {
        return this.errors.some(
          (e) => e.node.type === "trigger" && e.node.index === idx,
        )
          ? "border-red-400"
          : "";
      },
      getStepErrorClass(idx) {
        return this.errors.some(
          (e) => e.node.type === "step" && e.node.index === idx,
        )
          ? "border-red-400"
          : "";
      },

      toggleCodeView() {
        if (this.codeView) {
          try {
            this.parseYamlToState(this.yamlText);
            this.codeView = false;
            this.validate();
          } catch (e) {
            alert(
              "YAML syntax error. Fix errors before switching back to visual mode.\n" +
                e.message,
            );
          }
        } else {
          this.yamlText = this.stateToYaml();
          this.codeView = true;
        }
      },

      markDirty() {
        if (!this.dirty) {
          this.dirty = true;
        }
        this.startAutoSave();
      },
      startAutoSave() {
        clearTimeout(this.autoSaveTimer);
        this.autoSaveTimer = setTimeout(() => this.saveDraft(), 30000);
      },

      async saveDraft() {
        const yaml = this.stateToYaml();
        try {
          const resp = await fetch("/service/web/pipelines/" + this.name, {
            method: "PUT",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ yaml, version: this.version }),
          });
          if (resp.status === 409) {
            alert(
              "This draft was modified elsewhere. Please refresh the page.",
            );
            return;
          }
          const data = await resp.json();
          this.version = data.version;
          this.status = data.status;
          this.dirty = false;
        } catch (e) {
          console.error("Auto-save failed:", e);
        }
      },

      async publish() {
        if (this.publishDisabled) return;
        await this.saveDraft();
        try {
          const resp = await fetch(
            "/service/web/pipelines/" + this.name + "/publish",
            {
              method: "PUT",
              headers: { "Content-Type": "application/json" },
              body: JSON.stringify({ version: this.version }),
            },
          );
          if (resp.status === 409) {
            alert(
              "This draft was modified elsewhere. Please refresh the page.",
            );
            return;
          }
          const data = await resp.json();
          this.version = data.version;
          this.status = "published";
        } catch (e) {
          console.error("Publish failed:", e);
          alert("Publish failed: " + e.message);
        }
      },

      async loadMockPayload() {
        try {
          const resp = await fetch(
            "/service/web/pipelines/" +
              this.name +
              "/mock?source=" +
              this.testTriggerSource,
          );
          const data = await resp.json();
          this.testMockPayload = JSON.stringify(data.payload, null, 2);
        } catch (e) {
          console.error("Failed to load mock payload:", e);
        }
      },

      async runTest() {
        await this.saveDraft();
        const upToIdx = this.selectedNode?.index;
        if (upToIdx === null || upToIdx === undefined) return;
        try {
          const resp = await fetch(
            "/service/web/pipelines/" + this.name + "/test",
            {
              method: "POST",
              headers: { "Content-Type": "application/json" },
              body: JSON.stringify({
                trigger_source: this.testTriggerSource,
                mock_payload: JSON.parse(this.testMockPayload || "{}"),
                up_to_step_index: upToIdx,
              }),
            },
          );
          this.testResults = await resp.json();
        } catch (e) {
          console.error("Test failed:", e);
          this.testResults = { success: false, error: e.message };
        }
      },
    }));
  }

  if (window.Alpine) {
    register();
  } else {
    document.addEventListener("alpine:init", register);
  }
})();
