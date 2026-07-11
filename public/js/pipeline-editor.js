(function () {
  function register() {
    Alpine.data('pipelineEditor', () => ({
      name: '',
      description: '',
      status: 'draft',
      version: 1,
      dirty: false,
      undoStack: [],
      redoStack: [],
      triggers: [],
      steps: [],
      selectedNode: null,
      drawerOpen: false,
      drawerExpanded: false,
      drawerTab: 'setup',
      drawerDirty: false,
      drawerSnapshot: null,
      codeView: false,
      yamlText: '',
      variablePickerOpen: false,
      variablePickerTarget: null,
      variablePickerSource: 'event',
      errors: [],
      publishDisabled: false,
      autoSaveTimer: null,
      testTriggerSource: 'event',
      testMockPayload: '{}',
      testResults: null,
      capabilities: [],
      defaultTemplateSet: null,
      loading: false,
      saving: false,
      testing: false,
      publishing: false,
      dragFromIdx: null,
      dragOverIdx: null,
      historyOpen: false,
      versions: [],
      selectedVersion: null,
      selectedVersionYaml: '',
      historyLoading: false,
      compareMode: false,
      compareLeft: null,
      compareRight: null,
      diffResult: null,

      init() {
        const el = this.$el;
        const pipelineName = el.dataset.pipelineName || '';
        this.name = pipelineName;
        if (pipelineName) this.loadPipeline(pipelineName);
        this.fetchCapabilities();
        this.pushUndo();
        this.loadVersions();
      },

      async loadPipeline(pipelineName) {
        this.loading = true;
        try {
          const resp = await fetch(
            `/service/web/pipelines/${pipelineName}/yaml`,
          );
          const data = await resp.json();
          this.version = data.version;
          this.status = data.status;
          if (data.yaml) this.parseYamlToState(data.yaml);
        } catch (e) {
          console.error('Failed to load pipeline:', e);
          showToast('Failed to load pipeline', 'error');
        } finally {
          this.loading = false;
        }
      },

      async fetchCapabilities() {
        try {
          const resp = await fetch('/service/web/pipelines/capabilities');
          const json = await resp.json();
          this.capabilities = json.data || [];
          const set = new Set();
          for (const cap of this.capabilities) {
            for (const op of cap.operations || []) {
              if (op.input && op.input.length > 0) {
                set.add(this.buildParamsTemplate(op.input));
              }
            }
          }
          set.add('{}');
          this.defaultTemplateSet = set;
        } catch (e) {
          console.error('Failed to load capabilities:', e);
        }
      },

      getOperationsFor(capType) {
        const cap = this.capabilities.find((c) => c.type === capType);
        return cap ? cap.operations || [] : [];
      },

      getOperation(capType, opName) {
        const ops = this.getOperationsFor(capType);
        return ops.find((o) => o.name === opName) || null;
      },

      typeDefaultValue(type) {
        switch (type) {
          case 'string':
            return '<string>';
          case 'int':
          case 'int64':
            return 0;
          case 'bool':
            return false;
          case '[]string':
            return [];
          case 'map[string]any':
            return {};
          default:
            console.warn('Unknown ParamDef type:', type);
            return '<string>';
        }
      },

      getDefaultParams(capType, opName) {
        const op = this.getOperation(capType, opName);
        if (!op || !op.input || op.input.length === 0) {
          return '{}';
        }
        return this.buildParamsTemplate(op.input);
      },

      buildParamsTemplate(input) {
        const obj = {};
        for (const p of input) {
          obj[p.name] = this.typeDefaultValue(p.type);
        }
        return JSON.stringify(obj, null, 2);
      },

      isParamsDefault(paramsText) {
        if (!paramsText) return true;
        const trimmed = paramsText.trim();
        if (this.defaultTemplateSet && this.defaultTemplateSet.has(trimmed)) {
          return true;
        }
        for (const cap of this.capabilities) {
          for (const op of cap.operations || []) {
            if (this.getDefaultParams(cap.type, op.name).trim() === trimmed) {
              return true;
            }
          }
        }
        return false;
      },

      onCapabilityChange(idx) {
        const capType = this.steps[idx].capability;
        const wasDefault = this.isParamsDefault(this.steps[idx].paramsText);
        const firstOp = this.getOperationsFor(capType)[0];
        this.steps[idx].operation = firstOp ? firstOp.name : '';
        if (wasDefault && this.steps[idx].operation) {
          this.steps[idx].paramsText = this.getDefaultParams(
            capType,
            this.steps[idx].operation,
          );
        }
        this.drawerDirty = true;
      },

      onOperationChange(idx) {
        const step = this.steps[idx];
        step.paramsText = this.getDefaultParams(
          step.capability,
          step.operation,
        );
        this.drawerDirty = true;
      },

      getCurrentOperationInput(idx) {
        const step = this.steps[idx];
        if (!step || !step.capability || !step.operation) return [];
        const op = this.getOperation(step.capability, step.operation);
        return op ? op.input || [] : [];
      },

      getEventsForTrigger() {
        const groups = [];
        for (const cap of this.capabilities) {
          if (cap.events && cap.events.length > 0) {
            groups.push({
              capability: cap.type,
              description: cap.description,
              events: cap.events,
            });
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
          this.description = obj.description || '';
          this.triggers = (obj.triggers || []).map((t) => ({
            type: t.type || 'event',
            enabled: t.enabled !== false,
            event: t.event || '',
            cron: t.cron || '',
            webhook: t.webhook || {
              path: '',
              method: 'POST',
              auth: { token: '', hmac_secret: '' },
            },
          }));
          this.steps = (obj.steps || []).map((s) => ({
            name: s.name || '',
            capability: s.capability || '',
            operation: s.operation || '',
            paramsText: JSON.stringify(s.params || {}, null, 2),
          }));
          this.validate();
        } catch (e) {
          console.error('YAML parse error:', e);
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
            if (t.type === 'event') e.event = t.event;
            if (t.type === 'cron') e.cron = t.cron;
            if (t.type === 'webhook') e.webhook = t.webhook;
            return e;
          }),
          steps: this.steps.map((s) => ({
            name: s.name,
            capability: s.capability,
            operation: s.operation,
            params: (() => {
              try {
                return JSON.parse(s.paramsText || '{}');
              } catch {
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
          type: 'event',
          enabled: true,
          event: '',
          cron: '',
          webhook: {
            path: '',
            method: 'POST',
            auth: { token: '', hmac_secret: '' },
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

      confirmRemoveTrigger(idx) {
        var self = this;
        showConfirmModal({
          title: 'Remove Trigger',
          message: 'Remove this trigger from the pipeline?',
          confirmText: 'Remove',
          confirmClass: 'btn-error',
          onConfirm: function () {
            self.removeTrigger(idx);
          },
        });
      },

      addStep(afterIdx) {
        this.pushUndo();
        this.steps.splice(afterIdx, 0, {
          name: '',
          capability: '',
          operation: '',
          paramsText: '{}',
        });
        this.markDirty();
        this.selectNode('step', afterIdx);
      },

      removeStep(idx) {
        this.pushUndo();
        this.steps.splice(idx, 1);
        this.markDirty();
        this.validate();
        if (this.drawerOpen && this.selectedNode?.index === idx)
          this.drawerOpen = false;
      },

      confirmRemoveStep(idx) {
        var self = this;
        showConfirmModal({
          title: 'Delete Step',
          message: 'Delete this step from the pipeline?',
          confirmText: 'Delete',
          confirmClass: 'btn-error',
          onConfirm: function () {
            self.removeStep(idx);
          },
        });
      },

      duplicateStep(idx) {
        this.pushUndo();
        const copy = JSON.parse(JSON.stringify(this.steps[idx]));
        copy.name = copy.name + '-copy';
        this.steps.splice(idx + 1, 0, copy);
        this.markDirty();
      },

      dependsOnStep(step, targetIdx) {
        const re = /\{\{steps\.(\w+)\./g;
        const refs = [...(step.paramsText || '').matchAll(re)].map((m) => m[1]);
        return refs.some(
          (ref) => this.steps.findIndex((s) => s.name === ref) >= targetIdx,
        );
      },

      selectNode(type, idx) {
        if (this.drawerDirty && this.selectedNode) {
          var self = this;
          showConfirmModal({
            title: 'Discard Changes',
            message: 'You have unsaved changes. Discard them?',
            confirmText: 'Discard',
            confirmClass: 'btn-error',
            onConfirm: function () {
              self.restoreDrawerSnapshot();
              self.openDrawerNode(type, idx);
            },
          });
          return;
        }
        this.openDrawerNode(type, idx);
      },

      captureDrawerSnapshot(type, idx) {
        if (type === 'step') {
          return JSON.parse(JSON.stringify(this.steps[idx]));
        }
        if (type === 'trigger') {
          return JSON.parse(JSON.stringify(this.triggers[idx]));
        }
        return null;
      },

      restoreDrawerSnapshot() {
        if (!this.selectedNode || !this.drawerSnapshot) return;
        const { type, index } = this.selectedNode;
        if (type === 'step') {
          this.steps[index] = JSON.parse(JSON.stringify(this.drawerSnapshot));
        } else if (type === 'trigger') {
          this.triggers[index] = JSON.parse(
            JSON.stringify(this.drawerSnapshot),
          );
        }
        this.validate();
      },

      openDrawerNode(type, idx) {
        this.selectedNode = { type, index: idx };
        this.drawerOpen = true;
        this.drawerDirty = false;
        this.drawerTab = 'setup';
        this.drawerSnapshot = this.captureDrawerSnapshot(type, idx);
      },

      finishDrawerSession() {
        this.drawerDirty = false;
        this.drawerSnapshot = null;
        this.drawerOpen = false;
        this.selectedNode = null;
      },

      async saveDrawer() {
        if (!this.selectedNode) return;
        const { type, index } = this.selectedNode;
        if (type === 'step') {
          try {
            JSON.parse(this.steps[index].paramsText || '{}');
          } catch (e) {
            showToast('Invalid params JSON: ' + e.message, 'error');
            return;
          }
        }
        this.validate();
        const nodeErrors = this.errors.filter(
          (e) => e.node.type === type && e.node.index === index,
        );
        if (nodeErrors.length > 0) {
          showToast(nodeErrors[0].message, 'error');
          return;
        }
        if (this.drawerDirty) {
          this.pushUndo();
          this.markDirty();
        }
        this.finishDrawerSession();
        await this.saveDraft();
      },

      closeDrawer() {
        if (this.drawerDirty) {
          var self = this;
          showConfirmModal({
            title: 'Discard Changes',
            message: 'You have unsaved changes. Discard them?',
            confirmText: 'Discard',
            confirmClass: 'btn-error',
            onConfirm: function () {
              self.restoreDrawerSnapshot();
              self.finishDrawerSession();
            },
          });
          return;
        }
        this.finishDrawerSession();
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
        const template = '{{' + path + '}}';
        const textarea = document.querySelector(
          '[data-testid="params-editor"]',
        );
        if (textarea) {
          const start = textarea.selectionStart;
          const end = textarea.selectionEnd;
          step.paramsText =
            (step.paramsText || '').substring(0, start) +
            template +
            (step.paramsText || '').substring(end);
          setTimeout(() => {
            textarea.focus();
            textarea.setSelectionRange(
              start + template.length,
              start + template.length,
            );
          }, 50);
        } else {
          step.paramsText = (step.paramsText || '') + template;
        }
        this.drawerDirty = true;
        this.variablePickerOpen = false;
      },

      validate() {
        this.errors = [];
        if (this.triggers.filter((t) => t.enabled).length === 0)
          this.errors.push({
            node: { type: 'trigger', index: -1 },
            message: 'At least one trigger must be enabled',
          });
        if (this.steps.length === 0)
          this.errors.push({
            node: { type: 'step', index: -1 },
            message: 'At least one step is required',
          });
        for (let i = 0; i < this.triggers.length; i++) {
          const t = this.triggers[i];
          if (!t.enabled) continue;
          if (t.type === 'event' && !t.event)
            this.errors.push({
              node: { type: 'trigger', index: i },
              message: 'Event type is required',
            });
          if (t.type === 'webhook' && (!t.webhook || !t.webhook.path))
            this.errors.push({
              node: { type: 'trigger', index: i },
              message: 'Webhook path is required',
            });
          if (
            t.type === 'webhook' &&
            t.webhook &&
            !t.webhook.auth.token &&
            !t.webhook.auth.hmac_secret
          )
            this.errors.push({
              node: { type: 'trigger', index: i },
              message: 'At least one auth method is required',
            });
          if (t.type === 'cron' && !t.cron)
            this.errors.push({
              node: { type: 'trigger', index: i },
              message: 'Cron expression is required',
            });
        }
        for (let i = 0; i < this.steps.length; i++) {
          const s = this.steps[i];
          if (!s.name)
            this.errors.push({
              node: { type: 'step', index: i },
              message: 'Step name is required',
            });
          if (!s.capability)
            this.errors.push({
              node: { type: 'step', index: i },
              message: 'Capability is required',
            });
          if (!s.operation)
            this.errors.push({
              node: { type: 'step', index: i },
              message: 'Operation is required',
            });
          const re = /\{\{steps\.(\w+)\./g;
          const refs = [...(s.paramsText || '').matchAll(re)].map((m) => m[1]);
          for (const ref of refs) {
            const ri = this.steps.findIndex((ss) => ss.name === ref);
            if (ri === -1)
              this.errors.push({
                node: { type: 'step', index: i },
                message:
                  'Upstream variable {{steps.' +
                  ref +
                  '.*}} is invalid or has been removed',
              });
            else if (ri >= i)
              this.errors.push({
                node: { type: 'step', index: i },
                message:
                  'Depends on [' + ref + '] which must be above this step',
              });
          }
        }
        this.publishDisabled = this.errors.length > 0;
      },

      getTriggerErrorClass(idx) {
        return this.errors.some(
          (e) => e.node.type === 'trigger' && e.node.index === idx,
        )
          ? 'border-red-400'
          : '';
      },
      getStepErrorClass(idx) {
        return this.errors.some(
          (e) => e.node.type === 'step' && e.node.index === idx,
        )
          ? 'border-red-400'
          : '';
      },

      toggleCodeView() {
        if (this.codeView) {
          try {
            this.parseYamlToState(this.yamlText);
            this.codeView = false;
            this.validate();
          } catch (e) {
            showToast(
              'YAML syntax error. Fix errors before switching back to visual mode.\n' +
                e.message,
              'error',
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
        this.saving = true;
        const yaml = this.stateToYaml();
        try {
          const resp = await fetch('/service/web/pipelines/' + this.name, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ yaml, version: this.version }),
          });
          if (resp.status === 409) {
            showToast(
              'This draft was modified elsewhere. Please refresh the page.',
              'error',
            );
            return;
          }
          const data = await resp.json();
          this.version = data.version;
          this.status = data.status;
          this.dirty = false;
          showToast('Draft saved', 'success');
        } catch (e) {
          console.error('Auto-save failed:', e);
        } finally {
          this.saving = false;
        }
      },

      async publish() {
        if (this.publishDisabled) return;
        this.publishing = true;
        await this.saveDraft();
        try {
          const resp = await fetch(
            '/service/web/pipelines/' + this.name + '/publish',
            {
              method: 'PUT',
              headers: { 'Content-Type': 'application/json' },
              body: JSON.stringify({ version: this.version }),
            },
          );
          if (resp.status === 409) {
            showToast(
              'This draft was modified elsewhere. Please refresh the page.',
              'error',
            );
            return;
          }
          const data = await resp.json();
          this.version = data.version;
          this.status = 'published';
          showToast('Pipeline published', 'success');
        } catch (e) {
          console.error('Publish failed:', e);
          showToast('Publish failed: ' + e.message, 'error');
        } finally {
          this.publishing = false;
        }
      },

      async loadMockPayload() {
        try {
          const resp = await fetch(
            '/service/web/pipelines/' +
              this.name +
              '/mock?source=' +
              this.testTriggerSource,
          );
          const data = await resp.json();
          this.testMockPayload = JSON.stringify(data.payload, null, 2);
        } catch (e) {
          console.error('Failed to load mock payload:', e);
        }
      },

      async runTest() {
        await this.saveDraft();
        const upToIdx = this.selectedNode?.index;
        if (upToIdx === null || upToIdx === undefined) return;
        this.testing = true;
        try {
          const resp = await fetch(
            '/service/web/pipelines/' + this.name + '/test',
            {
              method: 'POST',
              headers: { 'Content-Type': 'application/json' },
              body: JSON.stringify({
                trigger_source: this.testTriggerSource,
                mock_payload: JSON.parse(this.testMockPayload || '{}'),
                up_to_step_index: upToIdx,
              }),
            },
          );
          this.testResults = await resp.json();
        } catch (e) {
          console.error('Test failed:', e);
          this.testResults = { success: false, error: e.message };
          showToast('Test failed: ' + e.message, 'error');
        } finally {
          this.testing = false;
        }
      },

      onStepDragStart(idx, e) {
        this.dragFromIdx = idx;
        e.dataTransfer.effectAllowed = 'move';
        e.dataTransfer.setData('text/plain', String(idx));
        e.target.closest('[data-sort-idx]').classList.add('opacity-50');
      },

      onStepDragEnd(e) {
        this.dragFromIdx = null;
        this.dragOverIdx = null;
        e.target.closest('[data-sort-idx]')?.classList.remove('opacity-50');
        this.$el
          .querySelectorAll('.drag-over-highlight')
          .forEach(function (el) {
            el.classList.remove(
              'drag-over-highlight',
              'border-t-2',
              'border-primary',
            );
          });
      },

      onStepDragOver(idx, e) {
        e.preventDefault();
        if (idx === this.dragFromIdx) return;
        e.dataTransfer.dropEffect = 'move';
        this.dragOverIdx = idx;
        var stepEl = e.currentTarget.closest('[data-sort-idx]');
        if (stepEl) {
          this.$el
            .querySelectorAll('.drag-over-highlight')
            .forEach(function (el) {
              el.classList.remove(
                'drag-over-highlight',
                'border-t-2',
                'border-primary',
              );
            });
          stepEl.classList.add(
            'drag-over-highlight',
            'border-t-2',
            'border-primary',
          );
        }
      },

      onStepDragLeave(e) {
        var stepEl = e.currentTarget.closest('[data-sort-idx]');
        if (stepEl) {
          stepEl.classList.remove(
            'drag-over-highlight',
            'border-t-2',
            'border-primary',
          );
        }
      },

      onStepDrop(idx, e) {
        e.preventDefault();
        this.dragOverIdx = null;
        this.$el
          .querySelectorAll('.drag-over-highlight')
          .forEach(function (el) {
            el.classList.remove(
              'drag-over-highlight',
              'border-t-2',
              'border-primary',
            );
          });
        if (this.dragFromIdx === null || this.dragFromIdx === idx) return;

        if (
          this.dependsOnStep(
            this.steps[this.dragFromIdx],
            Math.min(idx, this.dragFromIdx),
          )
        ) {
          showToast(
            'Cannot move: this step depends on data from a step at or above the target position.',
            'warning',
          );
          return;
        }

        this.pushUndo();
        var item = this.steps.splice(this.dragFromIdx, 1)[0];
        this.steps.splice(idx, 0, item);
        this.markDirty();
        this.validate();
        this.dragFromIdx = null;
      },

      downloadYaml() {
        var yaml = this.stateToYaml();
        var blob = new Blob([yaml], { type: 'application/x-yaml' });
        var url = URL.createObjectURL(blob);
        var a = document.createElement('a');
        a.href = url;
        a.download = (this.name || 'pipeline') + '.yaml';
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
      },

      triggerImport() {
        this.$el.querySelector('#yaml-import-input').click();
      },

      async handleYamlImport(e) {
        var file = e.target.files[0];
        if (!file) return;
        try {
          var text = await new Promise(function (resolve, reject) {
            var reader = new FileReader();
            reader.addEventListener('load', function (ev) {
              resolve(ev.target.result);
            });
            reader.addEventListener('error', function (ev) {
              reject(ev);
            });
            reader.readAsText(file);
          });
          var obj = jsyaml.load(text);
          if (!obj || typeof obj !== 'object') {
            showToast('Invalid YAML: not a pipeline definition', 'error');
            return;
          }
          this.pushUndo();
          this.parseYamlToState(text);
          this.markDirty();
          this.validate();
          showToast('YAML imported successfully', 'success');
        } catch (err) {
          showToast('Import failed: ' + err.message, 'error');
        } finally {
          e.target.value = '';
        }
      },

      async loadVersions() {
        this.historyLoading = true;
        try {
          var resp = await fetch(
            '/service/web/pipelines/' + this.name + '/versions',
          );
          if (!resp.ok) {
            this.versions = [];
            return;
          }
          this.versions = await resp.json();
        } catch (e) {
          console.error('Failed to load versions:', e);
          this.versions = [];
        } finally {
          this.historyLoading = false;
        }
      },

      toggleHistory() {
        this.historyOpen = !this.historyOpen;
        if (this.historyOpen && this.versions.length === 0) {
          this.loadVersions();
        }
      },

      async selectVersion(v) {
        this.selectedVersion = v;
        this.historyLoading = true;
        try {
          var resp = await fetch(
            '/service/web/pipelines/' + this.name + '/versions/' + v.version,
          );
          if (!resp.ok) throw new Error('Not found');
          var data = await resp.json();
          this.selectedVersionYaml = data.yaml;
        } catch (e) {
          console.error('Failed to load version:', e);
          this.selectedVersionYaml = '';
        } finally {
          this.historyLoading = false;
        }
      },

      relativeTime(isoStr) {
        var d = new Date(isoStr);
        var now = new Date();
        var diff = now - d;
        var mins = Math.floor(diff / 60000);
        if (mins < 60) return mins + ' minutes ago';
        var hours = Math.floor(mins / 60);
        if (hours < 24) return hours + ' hours ago';
        var days = Math.floor(hours / 24);
        return days + ' days ago';
      },

      toggleCompareMode() {
        this.compareMode = !this.compareMode;
        if (!this.compareMode) {
          this.compareLeft = null;
          this.compareRight = null;
          this.diffResult = null;
        }
      },

      toggleCompareVersion(v) {
        if (this.compareLeft && this.compareLeft.version === v.version) {
          this.compareLeft = null;
        } else if (
          this.compareRight &&
          this.compareRight.version === v.version
        ) {
          this.compareRight = null;
        } else if (!this.compareLeft) {
          this.compareLeft = v;
        } else if (!this.compareRight) {
          this.compareRight = v;
        }
        if (this.compareLeft && this.compareRight) {
          this.computeDiff();
        }
      },

      async computeDiff() {
        var left = this.compareLeft;
        var right = this.compareRight;
        var self = this;
        var fetchYaml = async function (v) {
          var resp = await fetch(
            '/service/web/pipelines/' + self.name + '/versions/' + v.version,
          );
          var data = await resp.json();
          return data.yaml || '';
        };

        try {
          var leftYaml = await fetchYaml(left);
          var rightYaml = await fetchYaml(right);
          var changes = Diff.diffLines(leftYaml || '', rightYaml || '');
          this.diffResult = changes.map(function (part) {
            return {
              text: part.value,
              added: part.added,
              removed: part.removed,
            };
          });
        } catch (e) {
          console.error('Diff error:', e);
          this.diffResult = null;
        }
      },
    }));
  }

  if (window.Alpine) {
    register();
  } else {
    document.addEventListener('alpine:init', register);
  }
})();
