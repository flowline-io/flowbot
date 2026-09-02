(function () {
  function normalizeWebhookConfig(webhook) {
    var src = webhook || {};
    var auth = src.auth || {};
    return {
      path: src.path || '',
      method: src.method || 'POST',
      auth: {
        token: auth.token || '',
        hmac_secret: auth.hmac_secret || '',
      },
    };
  }

  function generateWebhookToken() {
    var bytes = new Uint8Array(16);
    if (typeof crypto !== 'undefined' && crypto.getRandomValues) {
      crypto.getRandomValues(bytes);
    } else {
      for (var i = 0; i < bytes.length; i++) {
        bytes[i] = Math.floor(Math.random() * 256);
      }
    }
    return Array.prototype.map
      .call(bytes, function (b) {
        return b.toString(16).padStart(2, '0');
      })
      .join('');
  }

  function register() {
    Alpine.data('pipelineEditor', () => ({
      pipelineURL(suffix) {
        const base = '/service/web/pipelines/' + encodeURIComponent(this.name);
        return suffix ? base + suffix : base;
      },
      name: '',
      description: '',
      renaming: false,
      renameValue: '',
      renamingBusy: false,
      enabled: true,
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
      drawerStepIndex: -1,
      drawerStep: null,
      drawerStepFormReady: false,
      codeView: false,
      yamlText: '',
      variablePickerOpen: false,
      variablePickerTarget: null,
      paramsAdvancedOpen: false,
      paramFieldErrors: {},
      variablePickerSource: 'event',
      errors: [],
      publishDisabled: false,
      autoSaveTimer: null,
      testTriggerSource: 'event',
      testMockPayload: '{}',
      testResults: null,
      capabilities: [],
      agentRunOptions: { tools: [], skills: [] },
      functionInvokeOptions: { functions: [] },
      memoryModalOpen: false,
      memoryKeys: [],
      memorySelectedKey: '',
      memoryContent: '',
      memoryPinned: false,
      memoryError: '',
      memorySaving: false,
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
        this.fetchAgentRunOptions();
        this.fetchFunctionInvokeOptions();
        this.pushUndo();
        this.loadVersions();
      },

      async loadPipeline(_pipelineName) {
        this.loading = true;
        try {
          const resp = await fetch(this.pipelineURL('/yaml'));
          if (!resp.ok) {
            throw new Error('HTTP ' + resp.status);
          }
          const data = await resp.json();
          this.version = data.version;
          this.status = data.status;
          if (data.yaml) this.parseYamlToState(data.yaml);
        } catch (e) {
          console.error('Failed to load pipeline:', e);
          showToast(
            flowbotI18n(
              'client.pipeline.load_failed',
              'Failed to load pipeline',
            ),
            'error',
          );
        } finally {
          this.loading = false;
        }
      },

      startRename() {
        this.renameValue = this.name;
        this.renaming = true;
        setTimeout(() => {
          const input = document.querySelector(
            '[data-testid="input-rename-pipeline"]',
          );
          if (input) {
            input.focus();
            input.select();
          }
        }, 0);
      },

      cancelRename() {
        if (this.renamingBusy) return;
        this.renaming = false;
        this.renameValue = this.name;
      },

      async confirmRename() {
        if (!this.renaming || this.renamingBusy) return;
        const nextName = (this.renameValue || '').trim();
        if (!nextName) {
          showToast(
            flowbotI18n(
              'client.pipeline.name_required',
              'Pipeline name is required',
            ),
            'error',
          );
          this.renameValue = this.name;
          this.renaming = false;
          return;
        }
        if (nextName === this.name) {
          this.renaming = false;
          return;
        }
        this.renamingBusy = true;
        try {
          const resp = await fetch(this.pipelineURL('/rename'), {
            method: 'PUT',
            headers: flowbotCSRFHeaders({ 'Content-Type': 'application/json' }),
            body: JSON.stringify({ name: nextName }),
          });
          const data = await resp.json().catch(() => ({}));
          if (!resp.ok) {
            const message =
              (data.error && data.error.message) ||
              flowbotI18n(
                'client.pipeline.rename_failed',
                'Failed to rename pipeline',
              );
            showToast(message, 'error');
            this.renaming = false;
            this.renameValue = this.name;
            return;
          }
          const renamed = (data && data.name) || nextName;
          showToast(
            flowbotI18n('client.pipeline.renamed', 'Pipeline renamed'),
            'success',
          );
          window.location.href =
            '/service/web/pipelines/' + encodeURIComponent(renamed);
        } catch (e) {
          console.error('Failed to rename pipeline:', e);
          showToast(
            flowbotI18n(
              'client.pipeline.rename_failed',
              'Failed to rename pipeline',
            ),
            'error',
          );
          this.renaming = false;
          this.renameValue = this.name;
        } finally {
          this.renamingBusy = false;
        }
      },

      async fetchAgentRunOptions() {
        try {
          const resp = await fetch('/service/web/pipelines/agent-run-options');
          if (!resp.ok) {
            throw new Error('HTTP ' + resp.status);
          }
          const json = await resp.json();
          this.agentRunOptions = json.data || {
            tools: [],
            skills: [],
          };
        } catch (e) {
          console.error('Failed to load agent run options:', e);
          showToast(
            flowbotI18n(
              'client.pipeline.agent_options_failed',
              'Failed to load agent run options',
            ),
            'error',
          );
        }
      },

      async fetchFunctionInvokeOptions() {
        try {
          const resp = await fetch(
            '/service/web/pipelines/function-invoke-options',
          );
          if (!resp.ok) {
            throw new Error('HTTP ' + resp.status);
          }
          const json = await resp.json();
          this.functionInvokeOptions = json.data || { functions: [] };
          this.fillDrawerSelects();
        } catch (e) {
          console.error('Failed to load function invoke options:', e);
          showToast(
            flowbotI18n(
              'client.pipeline.function_options_failed',
              'Failed to load function options',
            ),
            'error',
          );
        }
      },

      async fetchCapabilities() {
        try {
          const resp = await fetch('/service/web/pipelines/capabilities');
          if (!resp.ok) {
            throw new Error('HTTP ' + resp.status);
          }
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
          this.fillDrawerSelects();
        } catch (e) {
          console.error('Failed to load capabilities:', e);
          showToast(
            flowbotI18n(
              'client.pipeline.capabilities_failed',
              'Failed to load capabilities',
            ),
            'error',
          );
        }
      },

      selectedStepIndex() {
        const node = this.selectedNode;
        if (!node || node.type !== 'step') return null;
        const idx = node.index;
        if (!Number.isInteger(idx) || idx < 0 || idx >= this.steps.length) {
          return null;
        }
        return idx;
      },

      selectedTriggerIndex() {
        const node = this.selectedNode;
        if (!node || node.type !== 'trigger') return null;
        const idx = node.index;
        if (!Number.isInteger(idx) || idx < 0 || idx >= this.triggers.length) {
          return null;
        }
        return idx;
      },

      selectedStep() {
        const idx = this.selectedStepIndex();
        if (idx == null) return null;
        return this.steps[idx] ?? null;
      },

      selectedTrigger() {
        const idx = this.selectedTriggerIndex();
        if (idx == null) return null;
        return this.triggers[idx] ?? null;
      },

      // CSP-safe: expose as a getter property for x-for (prefer `in enabledTriggers`, not a call).
      get enabledTriggers() {
        return this.triggers.filter(function (tr) {
          return tr.enabled;
        });
      },

      versionLabel(v) {
        if (!v || v.version == null) return '';
        return 'v' + v.version;
      },

      selectedVersionLabel() {
        return this.versionLabel(this.selectedVersion);
      },

      selectedVersionCreatedAt() {
        return this.selectedVersion ? this.selectedVersion.created_at : '';
      },

      selectedStepCapability() {
        const step = this.selectedStep();
        return step ? step.capability : '';
      },

      selectedStepOperations() {
        return this.getOperationsFor(this.selectedStepCapability());
      },

      drawerStepOperations() {
        if (!this.drawerStep) {
          return [];
        }
        return this.getOperationsFor(this.drawerStep.capability);
      },

      resetDrawerStep(idx) {
        const step = this.steps[idx];
        if (!step) {
          this.drawerStep = null;
          return;
        }
        this.drawerStep = {
          name: step.name || '',
          capability: step.capability || '',
          operation: step.operation || '',
          paramsText: step.paramsText || '{}',
        };
      },

      syncDrawerStepParamsText(idx) {
        if (idx !== this.drawerStepIndex || !this.drawerStep) {
          return;
        }
        const step = this.steps[idx];
        if (!step) {
          return;
        }
        this.drawerStep.paramsText = step.paramsText || '{}';
      },

      onDrawerParamsTextInput() {
        const idx = this.drawerStepIndex;
        if (idx < 0 || !this.drawerStep) {
          return;
        }
        const step = this.steps[idx];
        if (!step) {
          return;
        }
        step.paramsText = this.drawerStep.paramsText;
        this.onParamsTextInput(idx);
      },

      onDrawerStepNameInput() {
        const idx = this.drawerStepIndex;
        if (idx < 0 || !this.drawerStep) {
          return;
        }
        this.steps[idx].name = this.drawerStep.name;
        this.drawerDirty = true;
      },

      onDrawerCapabilityChange(value) {
        if (this.fillingDrawerSelects) {
          return;
        }
        const idx = this.drawerStepIndex;
        if (idx < 0 || !this.drawerStep) {
          return;
        }
        this.drawerStep.capability = value;
        this.steps[idx].capability = value;
        this.onCapabilityChange(idx);
        this.drawerStep.operation = this.steps[idx].operation;
        this.syncDrawerStepParamsText(idx);
        this.fillDrawerSelects();
      },

      onDrawerOperationChange(value) {
        if (this.fillingDrawerSelects) {
          return;
        }
        const idx = this.drawerStepIndex;
        if (idx < 0 || !this.drawerStep) {
          return;
        }
        this.drawerStep.operation = value;
        this.steps[idx].operation = value;
        this.onOperationChange(idx);
        this.syncDrawerStepParamsText(idx);
        this.fillDrawerSelects();
      },

      fillSelect(el, placeholder, items, selected) {
        if (!el) {
          return;
        }
        this.fillingDrawerSelects = true;
        try {
          const selectedStr =
            selected === undefined || selected === null ? '' : String(selected);
          const opts = [];
          const placeholderOpt = document.createElement('option');
          placeholderOpt.value = '';
          placeholderOpt.disabled = true;
          placeholderOpt.textContent = placeholder || '';
          opts.push(placeholderOpt);
          let found = false;
          for (let i = 0; i < items.length; i++) {
            const item = items[i];
            const opt = document.createElement('option');
            opt.value = item.value;
            opt.textContent = item.label;
            if (item.title) {
              opt.title = item.title;
            }
            if (item.value === selectedStr) {
              opt.selected = true;
              found = true;
            }
            opts.push(opt);
          }
          if (selectedStr && !found) {
            const orphan = document.createElement('option');
            orphan.value = selectedStr;
            orphan.textContent = selectedStr;
            orphan.selected = true;
            opts.push(orphan);
          }
          el.replaceChildren(...opts);
          el.value = selectedStr;
        } finally {
          this.fillingDrawerSelects = false;
        }
      },

      fillCapabilitySelect(el) {
        const items = (this.capabilities || []).map((cap) => ({
          value: cap.type,
          label: cap.type,
          title: cap.description || '',
        }));
        const selected = this.drawerStep ? this.drawerStep.capability : '';
        this.fillSelect(el, el.dataset.placeholder, items, selected);
      },

      fillOperationSelect(el) {
        const items = this.drawerStepOperations().map((op) => ({
          value: op.name,
          label: op.name,
          title: op.description || '',
        }));
        const selected = this.drawerStep ? this.drawerStep.operation : '';
        this.fillSelect(el, el.dataset.placeholder, items, selected);
      },

      fillFunctionNameSelect(el) {
        const idx = this.drawerStepIndex;
        const items = this.getFunctionInvokeNameOptions(idx).map((fn) => ({
          value: fn.name,
          label: this.functionInvokeNameLabel(fn.name, fn.orphaned),
        }));
        this.fillSelect(
          el,
          el.dataset.placeholder,
          items,
          this.getFunctionInvokeName(idx),
        );
      },

      fillFunctionVersionSelect(el) {
        const idx = this.drawerStepIndex;
        const items = this.getFunctionInvokeVersions(idx).map((ver) => ({
          value: String(ver),
          label: String(ver),
        }));
        this.fillSelect(
          el,
          el.dataset.placeholder,
          items,
          this.getFunctionInvokeVersion(idx),
        );
      },

      fillDrawerSelects() {
        if (!this.drawerStepFormReady || !this.drawerStep || !this.$el) {
          return;
        }
        const root = this.$el;
        const cap = root.querySelector('[data-testid="step-capability-select"]');
        if (cap) {
          this.fillCapabilitySelect(cap);
        }
        const op = root.querySelector('[data-testid="step-operation-select"]');
        if (op) {
          this.fillOperationSelect(op);
        }
        const name = root.querySelector(
          '[data-testid="function-invoke-name-select"]',
        );
        if (name) {
          this.fillFunctionNameSelect(name);
        }
        const ver = root.querySelector(
          '[data-testid="function-invoke-version-select"]',
        );
        if (ver) {
          this.fillFunctionVersionSelect(ver);
        }
      },

      isCapabilityInList(capType) {
        return (this.capabilities || []).some((cap) => cap.type === capType);
      },

      hasTestResultSteps() {
        return !!(this.testResults && this.testResults.steps);
      },

      formatTestStepOutput(output) {
        if (output === undefined || output === null) {
          return '';
        }
        try {
          return JSON.stringify(output, null, 2);
        } catch {
          return String(output);
        }
      },

      formatTestStepDuration(durationMs) {
        if (durationMs === undefined || durationMs === null || durationMs === '') {
          return '';
        }
        const num = Number(durationMs);
        if (!Number.isFinite(num)) {
          return '';
        }
        return num + 'ms';
      },

      priorStepIndexes() {
        const idx = this.selectedStepIndex();
        const n = idx == null ? 0 : idx;
        const out = [];
        for (let i = 0; i < n; i++) {
          out.push(i);
        }
        return out;
      },

      stepNameAt(idx) {
        const step = this.steps[idx];
        return step && step.name ? step.name : '';
      },

      stepVarPath(idx, suffix) {
        const name = this.stepNameAt(idx);
        if (!name) return '';
        return 'steps.' + name + '.' + suffix;
      },

      insertStepVariable(idx, suffix) {
        const path = this.stepVarPath(idx, suffix);
        if (path) this.insertVariable(path);
      },

      validateSelectedNode() {
        if (!this.selectedNode) return;
        const type = this.selectedNode.type;
        const list =
          type === 'step'
            ? this.steps
            : type === 'trigger'
              ? this.triggers
              : null;
        if (!list) {
          this.finishDrawerSession();
          return;
        }
        const idx = this.selectedNode.index;
        if (
          !Number.isInteger(idx) ||
          idx < 0 ||
          idx >= list.length ||
          list[idx] == null
        ) {
          this.finishDrawerSession();
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
          case 'number':
            return 0;
          case 'bool':
            return false;
          case '[]string':
          case '[]int64':
            return [];
          case 'map[string]any':
            return {};
          case 'any':
            return null;
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

      buildParamsTemplate(_input) {
        return '{}';
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

      formatStepParamsPreview(paramsText) {
        const trimmed = (paramsText || '').trim();
        if (!trimmed || trimmed === '{}') return '';
        try {
          return JSON.stringify(JSON.parse(trimmed), null, 2);
        } catch {
          return trimmed;
        }
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
        this.syncDrawerStepParamsText(idx);
        this.drawerDirty = true;
      },

      setStepCapability(idx, value) {
        const step = this.steps[idx];
        if (!step) {
          return;
        }
        step.capability = value;
        this.onCapabilityChange(idx);
      },

      setStepName(idx, value) {
        const step = this.steps[idx];
        if (!step) {
          return;
        }
        step.name = value;
        this.drawerDirty = true;
      },

      setStepParamsText(idx, value) {
        const step = this.steps[idx];
        if (!step) {
          return;
        }
        step.paramsText = value;
        this.syncDrawerStepParamsText(idx);
        this.onParamsTextInput(idx);
      },

      onOperationChange(idx) {
        const step = this.steps[idx];
        step.paramsText = this.getDefaultParams(
          step.capability,
          step.operation,
        );
        this.syncDrawerStepParamsText(idx);
        this.drawerDirty = true;
      },

      setStepOperation(idx, value) {
        const step = this.steps[idx];
        if (!step) {
          return;
        }
        step.operation = value;
        this.onOperationChange(idx);
      },

      getCurrentOperationInput(idx) {
        const step = this.steps[idx];
        if (!step || !step.capability || !step.operation) return [];
        const op = this.getOperation(step.capability, step.operation);
        return op ? op.input || [] : [];
      },

      getFormOperationInput(idx) {
        if (idx == null) return [];
        return this.getCurrentOperationInput(idx).filter((p) => {
          if (this.isAgentRunStringListParam(idx, p.name)) {
            return false;
          }
          if (this.isFunctionsInvokeReservedParam(idx, p.name)) {
            return false;
          }
          return true;
        });
      },

      isParamTypeString(p) {
        return p?.type === 'string';
      },

      isParamTypeNumber(p) {
        return (
          p?.type === 'int' ||
          p?.type === 'int64' ||
          p?.type === 'number'
        );
      },

      isParamTypeBool(p) {
        return p?.type === 'bool';
      },

      isParamTypeStringList(p) {
        return p?.type === '[]string';
      },

      isParamTypeIntList(p) {
        return p?.type === '[]int64';
      },

      isParamTypeMap(p) {
        return p?.type === 'map[string]any';
      },

      isParamTypeAny(p) {
        return p?.type === 'any';
      },

      // True when value is a pipeline {{...}} expression (not a schema default placeholder).
      isPipelineExpr(value) {
        return (
          typeof value === 'string' &&
          value.indexOf('{{') !== -1 &&
          value.indexOf('}}') !== -1
        );
      },

      numberParamPlaceholder(p) {
        if (p && p.required) {
          return flowbotI18n(
            'client.pipeline.number_param_required',
            '0 or {{expr}}',
          );
        }
        return flowbotI18n('client.pipeline.number_param_optional', 'optional');
      },

      paramFieldPlaceholder(p) {
        if (p && p.required) {
          return flowbotI18n(
            'client.pipeline.param_placeholder_required',
            'required',
          );
        }
        return flowbotI18n(
          'client.pipeline.param_placeholder_optional',
          'optional',
        );
      },

      errorSummaryText() {
        var count = this.errors.length;
        return flowbotI18n(
          'client.pipeline.error_summary',
          '{{.Count}} error(s). Publish is disabled.',
        ).replace('{{.Count}}', String(count));
      },

      triggerEventLabel(t) {
        var val =
          t && t.event
            ? t.event
            : flowbotI18n('client.pipeline.placeholder_ellipsis', '...');
        return flowbotI18n(
          'client.pipeline.trigger_event_prefix',
          'Event: {{.Value}}',
        ).replace('{{.Value}}', val);
      },

      triggerCronLabel(t) {
        var val =
          t && t.cron
            ? t.cron
            : flowbotI18n('client.pipeline.placeholder_ellipsis', '...');
        return flowbotI18n(
          'client.pipeline.trigger_cron_prefix',
          'Cron: {{.Value}}',
        ).replace('{{.Value}}', val);
      },

      stepDisplayName(step) {
        if (step && step.name) {
          return step.name;
        }
        return flowbotI18n('client.pipeline.unnamed_step', 'Unnamed Step');
      },

      extraFieldsHint(idx) {
        var count = this.getExtraParamKeys(idx).length;
        return flowbotI18n(
          'client.pipeline.extra_fields',
          '{{.Count}} extra field(s) only visible in Advanced JSON.',
        ).replace('{{.Count}}', String(count));
      },

      customEventLabel(eventName) {
        var suffix = flowbotI18n(
          'client.pipeline.custom_event_suffix',
          '(custom)',
        );
        return (eventName || '') + ' ' + suffix;
      },

      isParamTemplateValue(value, type) {
        if (this.isPipelineExpr(value)) {
          return false;
        }
        const def = this.typeDefaultValue(type);
        switch (type) {
          case 'string':
            return String(value).trim() === String(def).trim();
          case 'int':
          case 'int64':
          case 'number':
            return Number(value) === Number(def);
          case 'bool':
            return value === def;
          case '[]string':
          case '[]int64':
            return (
              Array.isArray(value) &&
              Array.isArray(def) &&
              value.length === 0 &&
              def.length === 0
            );
          case 'map[string]any':
            return (
              typeof value === 'object' &&
              value !== null &&
              !Array.isArray(value) &&
              Object.keys(value).length === 0
            );
          case 'any':
            return value === null || value === undefined;
          default:
            return false;
        }
      },

      isAgentRunStep(idx) {
        const step = this.steps[idx];
        return step?.capability === 'agent' && step?.operation === 'run';
      },

      isFunctionsInvokeStep(idx) {
        const step = this.steps[idx];
        return step?.capability === 'functions' && step?.operation === 'invoke';
      },

      pipelineMemoryEnabled() {
        if (!this.name) {
          return false;
        }
        const available = (this.agentRunOptions.tools || []).includes(
          'memory_set',
        );
        if (!available) {
          return false;
        }
        for (let i = 0; i < this.steps.length; i++) {
          if (!this.isAgentRunStep(i)) {
            continue;
          }
          const tools = this.getAgentRunParamList(i, 'tools');
          if (
            tools.includes('memory_set') ||
            tools.includes('memory_get') ||
            tools.includes('memory_list')
          ) {
            return true;
          }
        }
        return false;
      },

      parseStepParams(idx) {
        try {
          return JSON.parse(this.steps[idx]?.paramsText || '{}');
        } catch {
          return {};
        }
      },

      writeStepParams(idx, params) {
        const normalized = this.normalizeStepParams(idx, params);
        this.steps[idx].paramsText = JSON.stringify(normalized, null, 2);
        this.syncDrawerStepParamsText(idx);
        this.drawerDirty = true;
      },

      normalizeStepParams(idx, params) {
        const input = this.getCurrentOperationInput(idx);
        const normalized = { ...params };
        for (const p of input) {
          if (
            p.name in normalized &&
            this.isParamTemplateValue(normalized[p.name], p.type)
          ) {
            delete normalized[p.name];
          }
        }
        return normalized;
      },

      getParamDef(idx, name) {
        return (
          this.getCurrentOperationInput(idx).find((p) => p.name === name) ||
          null
        );
      },

      getStepParam(idx, name) {
        return this.parseStepParams(idx)[name];
      },

      shouldOmitParam(value, type, required) {
        if (required) {
          return false;
        }
        if (value === undefined || value === null) {
          return true;
        }
        if (this.isPipelineExpr(value)) {
          return false;
        }
        if (this.isParamTemplateValue(value, type)) {
          return true;
        }
        switch (type) {
          case 'string':
            return (
              String(value).trim() === '' || String(value).trim() === '<string>'
            );
          case 'int':
          case 'int64':
          case 'number':
            return value === '' || Number.isNaN(Number(value));
          case 'bool':
            return value === 'unset';
          case '[]string':
          case '[]int64':
            return !Array.isArray(value) || value.length === 0;
          case 'map[string]any':
            return (
              typeof value !== 'object' ||
              value === null ||
              Array.isArray(value) ||
              Object.keys(value).length === 0
            );
          case 'any':
            return value === null || value === undefined;
          default:
            return value === '' || value === undefined || value === null;
        }
      },

      coerceParamValue(value, type) {
        if (this.isPipelineExpr(value)) {
          return value;
        }
        switch (type) {
          case 'int':
          case 'number':
            return parseInt(value, 10);
          case 'int64':
            return Number(value);
          case 'bool':
            return value === true || value === 'true';
          default:
            return value;
        }
      },

      setStepParam(idx, name, value, type, required) {
        const pDef = this.getParamDef(idx, name);
        const req = required ?? pDef?.required ?? false;
        const params = this.parseStepParams(idx);
        if (this.shouldOmitParam(value, type, req)) {
          delete params[name];
        } else {
          params[name] = this.coerceParamValue(value, type);
        }
        this.writeStepParams(idx, params);
        const errKey = idx + ':' + name;
        if (this.paramFieldErrors[errKey]) {
          const next = { ...this.paramFieldErrors };
          delete next[errKey];
          this.paramFieldErrors = next;
        }
      },

      clearStepParam(idx, name) {
        const pDef = this.getParamDef(idx, name);
        if (!pDef || pDef.required) {
          return;
        }
        const params = this.parseStepParams(idx);
        delete params[name];
        this.writeStepParams(idx, params);
        const errKey = idx + ':' + name;
        if (this.paramFieldErrors[errKey]) {
          const next = { ...this.paramFieldErrors };
          delete next[errKey];
          this.paramFieldErrors = next;
        }
      },

      getStepParamString(idx, name) {
        const val = this.getStepParam(idx, name);
        if (val === undefined || val === null) {
          return '';
        }
        if (this.isParamTemplateValue(val, 'string')) {
          return '';
        }
        return String(val);
      },

      setStepParamString(idx, name, val) {
        const pDef = this.getParamDef(idx, name);
        this.setStepParam(idx, name, val, 'string', pDef?.required ?? false);
      },

      getStepParamNumber(idx, name) {
        const pDef = this.getParamDef(idx, name);
        const val = this.getStepParam(idx, name);
        if (val === undefined || val === null || val === '') {
          return '';
        }
        if (this.isPipelineExpr(val)) {
          return String(val);
        }
        if (this.isParamTemplateValue(val, pDef?.type || 'int')) {
          return '';
        }
        if (typeof val === 'number' && Number.isFinite(val)) {
          return String(val);
        }
        const num = Number(val);
        if (Number.isNaN(num)) {
          return '';
        }
        return String(val);
      },

      setStepParamNumber(idx, name, val, type) {
        const pDef = this.getParamDef(idx, name);
        const paramType = type || pDef?.type || 'int';
        if (val === '' || val === null || val === undefined) {
          this.setStepParam(idx, name, '', paramType, pDef?.required ?? false);
          return;
        }
        if (this.isPipelineExpr(val)) {
          this.setStepParam(
            idx,
            name,
            String(val),
            paramType,
            pDef?.required ?? false,
          );
          return;
        }
        this.setStepParam(idx, name, val, paramType, pDef?.required ?? false);
      },

      getStepParamBoolMode(idx, name) {
        const pDef = this.getParamDef(idx, name);
        const params = this.parseStepParams(idx);
        if (!(name in params)) {
          return 'unset';
        }
        if (
          !pDef?.required &&
          this.isParamTemplateValue(params[name], 'bool')
        ) {
          return 'unset';
        }
        return params[name] ? 'true' : 'false';
      },

      setStepParamBoolMode(idx, name, mode) {
        const pDef = this.getParamDef(idx, name);
        if (mode === 'unset') {
          this.setStepParam(
            idx,
            name,
            'unset',
            'bool',
            pDef?.required ?? false,
          );
          return;
        }
        this.setStepParam(
          idx,
          name,
          mode === 'true',
          'bool',
          pDef?.required ?? false,
        );
      },

      getStepParamStringList(idx, name) {
        const val = this.getStepParam(idx, name);
        if (!Array.isArray(val)) {
          return '';
        }
        return val.join(', ');
      },

      setStepParamStringList(idx, name, text) {
        const pDef = this.getParamDef(idx, name);
        const trimmed = (text || '').trim();
        if (!trimmed) {
          this.setStepParam(idx, name, [], '[]string', pDef?.required ?? false);
          return;
        }
        const values = trimmed
          .split(',')
          .map((item) => item.trim())
          .filter((item) => item.length > 0);
        this.setStepParam(
          idx,
          name,
          values,
          '[]string',
          pDef?.required ?? false,
        );
      },

      getStepParamIntList(idx, name) {
        const val = this.getStepParam(idx, name);
        if (this.isPipelineExpr(val)) {
          return String(val);
        }
        if (!Array.isArray(val)) {
          return '';
        }
        return val.join(', ');
      },

      setStepParamIntList(idx, name, text) {
        const pDef = this.getParamDef(idx, name);
        const required = pDef?.required ?? false;
        const trimmed = (text || '').trim();
        if (!trimmed) {
          this.setStepParam(idx, name, [], '[]int64', required);
          return;
        }
        if (this.isPipelineExpr(trimmed)) {
          this.setStepParam(idx, name, trimmed, '[]int64', required);
          return;
        }
        const values = [];
        const parts = trimmed.split(',');
        for (let i = 0; i < parts.length; i++) {
          const item = parts[i].trim();
          if (!item) {
            continue;
          }
          if (this.isPipelineExpr(item)) {
            values.push(item);
            continue;
          }
          const num = Number(item);
          if (Number.isNaN(num)) {
            continue;
          }
          values.push(num);
        }
        this.setStepParam(idx, name, values, '[]int64', required);
      },

      getStepParamMapJSON(idx, name) {
        const val = this.getStepParam(idx, name);
        if (val === undefined || val === null) {
          return '';
        }
        if (this.isParamTemplateValue(val, 'map[string]any')) {
          return '';
        }
        try {
          return JSON.stringify(val, null, 2);
        } catch {
          return '';
        }
      },

      setStepParamMapJSON(idx, name, text) {
        const pDef = this.getParamDef(idx, name);
        const required = pDef?.required ?? false;
        const trimmed = (text || '').trim();
        const errKey = idx + ':' + name;
        if (!trimmed || trimmed === '{}') {
          const next = { ...this.paramFieldErrors };
          delete next[errKey];
          this.paramFieldErrors = next;
          this.setStepParam(idx, name, {}, 'map[string]any', required);
          return;
        }
        try {
          const parsed = JSON.parse(trimmed);
          if (
            typeof parsed !== 'object' ||
            parsed === null ||
            Array.isArray(parsed)
          ) {
            throw new Error('must be a JSON object');
          }
          const next = { ...this.paramFieldErrors };
          delete next[errKey];
          this.paramFieldErrors = next;
          this.setStepParam(idx, name, parsed, 'map[string]any', required);
        } catch (e) {
          this.paramFieldErrors = {
            ...this.paramFieldErrors,
            [errKey]: e.message || 'Invalid JSON',
          };
          this.drawerDirty = true;
        }
      },

      getStepParamAnyJSON(idx, name) {
        const val = this.getStepParam(idx, name);
        if (val === undefined || val === null) {
          return '';
        }
        if (this.isParamTemplateValue(val, 'any')) {
          return '';
        }
        if (typeof val === 'string' && this.isPipelineExpr(val)) {
          return val;
        }
        try {
          return JSON.stringify(val, null, 2);
        } catch {
          return '';
        }
      },

      setStepParamAnyJSON(idx, name, text) {
        const pDef = this.getParamDef(idx, name);
        const required = pDef?.required ?? false;
        const trimmed = (text || '').trim();
        const errKey = idx + ':' + name;
        if (!trimmed) {
          const next = { ...this.paramFieldErrors };
          delete next[errKey];
          this.paramFieldErrors = next;
          this.setStepParam(idx, name, null, 'any', required);
          return;
        }
        if (this.isPipelineExpr(trimmed)) {
          const next = { ...this.paramFieldErrors };
          delete next[errKey];
          this.paramFieldErrors = next;
          this.setStepParam(idx, name, trimmed, 'any', required);
          return;
        }
        try {
          const parsed = JSON.parse(trimmed);
          const next = { ...this.paramFieldErrors };
          delete next[errKey];
          this.paramFieldErrors = next;
          this.setStepParam(idx, name, parsed, 'any', required);
        } catch (e) {
          this.paramFieldErrors = {
            ...this.paramFieldErrors,
            [errKey]: e.message || 'Invalid JSON',
          };
          this.drawerDirty = true;
        }
      },

      isParamFieldError(idx, name) {
        return Boolean(this.paramFieldErrors[idx + ':' + name]);
      },

      getExtraParamKeys(idx) {
        const params = this.parseStepParams(idx);
        const schemaNames = new Set(
          this.getCurrentOperationInput(idx).map((p) => p.name),
        );
        return Object.keys(params).filter((k) => !schemaNames.has(k));
      },

      isAgentRunStringListParam(idx, paramName) {
        return (
          this.isAgentRunStep(idx) &&
          (paramName === 'tools' || paramName === 'skills')
        );
      },

      isFunctionsInvokeReservedParam(idx, paramName) {
        return (
          this.isFunctionsInvokeStep(idx) &&
          (paramName === 'name' || paramName === 'version')
        );
      },

      getFunctionInvokeOption(name) {
        const functions = this.functionInvokeOptions.functions || [];
        return functions.find((item) => item.name === name) || null;
      },

      isFunctionInvokeNameExpr(idx) {
        return this.isPipelineExpr(this.getStepParam(idx, 'name'));
      },

      isFunctionInvokeVersionExpr(idx) {
        const val = this.getStepParam(idx, 'version');
        return this.isPipelineExpr(val);
      },

      showFunctionInvokeVersionSelect(idx) {
        if (
          this.isFunctionInvokeNameExpr(idx) ||
          this.isFunctionInvokeVersionExpr(idx)
        ) {
          return false;
        }
        return this.getFunctionInvokeVersions(idx).length > 0;
      },

      showFunctionInvokeVersionExpr(idx) {
        return (
          this.isFunctionInvokeNameExpr(idx) ||
          this.isFunctionInvokeVersionExpr(idx)
        );
      },

      functionInvokeNameLabel(name, orphaned) {
        if (!orphaned) {
          return name;
        }
        return flowbotI18n(
          'client.pipeline.function_unavailable',
          '{{.Name}} (unavailable)',
        ).replace('{{.Name}}', name);
      },

      getFunctionInvokeNameOptions(idx) {
        const functions = this.functionInvokeOptions.functions || [];
        const current = this.getFunctionInvokeName(idx);
        if (
          !current ||
          this.isFunctionInvokeNameExpr(idx) ||
          this.getFunctionInvokeOption(current)
        ) {
          return functions;
        }
        return [{ name: current, orphaned: true }, ...functions];
      },

      getSelectedFunctionInvokeOption(idx) {
        const name = this.getFunctionInvokeName(idx);
        if (!name || this.isFunctionInvokeNameExpr(idx)) {
          return null;
        }
        return this.getFunctionInvokeOption(name);
      },

      getFunctionInvokeVersions(idx) {
        const selected = this.getSelectedFunctionInvokeOption(idx);
        const versions = [...(selected?.published_versions || [])];
        if (this.isFunctionInvokeVersionExpr(idx)) {
          return versions;
        }
        const current = this.getStepParam(idx, 'version');
        if (
          current === undefined ||
          current === null ||
          current === '' ||
          this.isPipelineExpr(current)
        ) {
          return versions;
        }
        const versionNum = Number(current);
        if (Number.isFinite(versionNum) && !versions.includes(versionNum)) {
          versions.unshift(versionNum);
        }
        return versions;
      },

      getFunctionInvokeName(idx) {
        return this.getStepParamString(idx, 'name');
      },

      setFunctionInvokeName(idx, name) {
        if (this.fillingDrawerSelects) {
          return;
        }
        const option = this.getFunctionInvokeOption(name);
        this.setStepParamString(idx, 'name', name);
        if (!option) {
          return;
        }
        const currentVersion = this.getStepParam(idx, 'version');
        const versions = option.published_versions || [];
        const latest = option.latest_version || versions[0];
        const versionNum = Number(currentVersion);
        if (
          currentVersion === undefined ||
          currentVersion === null ||
          currentVersion === '' ||
          !versions.includes(versionNum)
        ) {
          if (latest !== undefined && latest !== null) {
            this.setStepParamNumber(idx, 'version', String(latest), 'number');
          }
        }
        this.fillDrawerSelects();
      },

      getFunctionInvokeVersion(idx) {
        const val = this.getStepParam(idx, 'version');
        if (val === undefined || val === null || val === '') {
          return '';
        }
        return String(val);
      },

      setFunctionInvokeVersion(idx, version) {
        if (this.fillingDrawerSelects) {
          return;
        }
        this.setStepParamNumber(idx, 'version', version, 'number');
      },

      setFunctionInvokeVersionExpr(idx, val) {
        this.setStepParamNumber(idx, 'version', val, 'number');
      },

      isParamValueMissing(val, type) {
        if (val === undefined || val === null) {
          return true;
        }
        if (this.isPipelineExpr(val)) {
          return false;
        }
        if (this.isParamTemplateValue(val, type)) {
          return true;
        }
        switch (type) {
          case 'string':
            return String(val).trim() === '';
          case 'int':
          case 'int64':
          case 'number':
            return val === '' || Number.isNaN(Number(val));
          case 'bool':
            return false;
          case '[]string':
          case '[]int64':
            return !Array.isArray(val) || val.length === 0;
          case 'map[string]any':
            return (
              typeof val !== 'object' ||
              val === null ||
              Array.isArray(val) ||
              Object.keys(val).length === 0
            );
          case 'any':
            return val === null || val === undefined;
          default:
            return (
              val === undefined || val === null || String(val).trim() === ''
            );
        }
      },

      validateStepParams(idx) {
        const step = this.steps[idx];
        if (!step) {
          return null;
        }
        try {
          JSON.parse(step.paramsText || '{}');
        } catch (e) {
          return flowbotI18n(
            'client.pipeline.invalid_params_json',
            'Invalid params JSON: {{.Error}}',
          ).replace('{{.Error}}', e.message);
        }
        const input = this.getCurrentOperationInput(idx);
        const params = this.parseStepParams(idx);
        for (const p of input) {
          if (p.required && this.isParamValueMissing(params[p.name], p.type)) {
            return flowbotI18n(
              'client.pipeline.param_required',
              'Parameter "{{.Name}}" is required',
            ).replace('{{.Name}}', p.name);
          }
          if (
            (p.type === 'map[string]any' || p.type === 'any') &&
            this.isParamFieldError(idx, p.name)
          ) {
            return flowbotI18n(
              'client.pipeline.param_invalid_json',
              'Parameter "{{.Name}}" has invalid JSON',
            ).replace('{{.Name}}', p.name);
          }
        }
        return null;
      },

      getAgentRunParamList(idx, key) {
        if (!this.isAgentRunStep(idx)) return [];
        const params = this.parseStepParams(idx);
        const value = params[key];
        return Array.isArray(value) ? value : [];
      },

      isAgentRunOptionSelected(idx, key, value) {
        return this.getAgentRunParamList(idx, key).includes(value);
      },

      setAgentRunParamList(idx, key, values) {
        const params = this.parseStepParams(idx);
        if (values.length === 0) {
          delete params[key];
        } else {
          params[key] = values;
        }
        this.writeStepParams(idx, params);
      },

      toggleAgentRunOption(idx, key, value) {
        const current = this.getAgentRunParamList(idx, key);
        const next = current.includes(value)
          ? current.filter((item) => item !== value)
          : [...current, value];
        this.setAgentRunParamList(idx, key, next);
      },

      onParamsTextInput(_idx) {
        this.drawerDirty = true;
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
          this.enabled = obj.enabled !== false;
          this.triggers = (obj.triggers || []).map((t) => ({
            type: t.type || 'event',
            enabled: t.enabled !== false,
            event: t.event || '',
            cron: t.cron || '',
            webhook: normalizeWebhookConfig(t.webhook),
          }));
          this.steps = (obj.steps || []).map((s) => ({
            name: s.name || '',
            capability: s.capability || '',
            operation: s.operation || '',
            paramsText: JSON.stringify(s.params || {}, null, 2),
          }));
          this.validate();
          this.validateSelectedNode();
        } catch (e) {
          console.error('YAML parse error:', e);
        }
      },

      stateToYaml() {
        const obj = {
          name: this.name,
          description: this.description,
          enabled: this.enabled !== false,
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
        this.validateSelectedNode();
      },

      redo() {
        if (this.redoStack.length === 0) return;
        const next = this.redoStack.pop();
        this.undoStack.push(JSON.parse(JSON.stringify(next)));
        this.triggers = JSON.parse(JSON.stringify(next.triggers));
        this.steps = JSON.parse(JSON.stringify(next.steps));
        this.markDirty();
        this.validate();
        this.validateSelectedNode();
      },

      addTrigger() {
        this.pushUndo();
        this.triggers.push({
          type: 'event',
          enabled: true,
          event: '',
          cron: '',
          webhook: normalizeWebhookConfig(null),
        });
        this.markDirty();
      },

      onTriggerTypeChange() {
        this.drawerDirty = true;
        var t = this.selectedTrigger();
        if (!t) {
          return;
        }
        if (t.type === 'webhook') {
          t.webhook = normalizeWebhookConfig(t.webhook);
          if (!t.webhook.auth.token && !t.webhook.auth.hmac_secret) {
            t.webhook.auth.token = generateWebhookToken();
          }
        }
      },

      syncDrawerAfterListRemoval(type, removedIdx) {
        if (!this.drawerOpen || this.selectedNode?.type !== type) return;
        const selIdx = this.selectedNode.index;
        if (selIdx === removedIdx) {
          this.finishDrawerSession();
          return;
        }
        if (selIdx > removedIdx) {
          const newIdx = selIdx - 1;
          this.selectedNode = { type, index: newIdx };
          this.drawerStepIndex = type === 'step' ? newIdx : -1;
          if (type === 'step') {
            this.resetDrawerStep(newIdx);
            this.fillDrawerSelects();
          } else {
            this.drawerStep = null;
            this.drawerStepFormReady = false;
          }
          this.drawerSnapshot = this.captureDrawerSnapshot(type, newIdx);
        }
      },

      removeTrigger(idx) {
        this.pushUndo();
        this.triggers.splice(idx, 1);
        this.markDirty();
        this.validate();
        this.syncDrawerAfterListRemoval('trigger', idx);
      },

      confirmRemoveTrigger(idx) {
        var self = this;
        showConfirmModal({
          title: flowbotI18n(
            'client.pipeline.remove_trigger.title',
            'Remove Trigger',
          ),
          message: flowbotI18n(
            'client.pipeline.remove_trigger.message',
            'Remove this trigger from the pipeline?',
          ),
          confirmText: flowbotI18n('client.pipeline.confirm_remove', 'Remove'),
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
        this.syncDrawerAfterListRemoval('step', idx);
      },

      confirmRemoveStep(idx) {
        var self = this;
        showConfirmModal({
          title: flowbotI18n(
            'client.pipeline.delete_step.title',
            'Delete Step',
          ),
          message: flowbotI18n(
            'client.pipeline.delete_step.message',
            'Delete this step from the pipeline?',
          ),
          confirmText: flowbotI18n('client.pipeline.confirm_delete', 'Delete'),
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
            title: flowbotI18n(
              'client.pipeline.discard_changes.title',
              'Discard Changes',
            ),
            message: flowbotI18n(
              'client.pipeline.discard_changes.message',
              'You have unsaved changes. Discard them?',
            ),
            confirmText: flowbotI18n(
              'client.pipeline.confirm_discard',
              'Discard',
            ),
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
          this.resetDrawerStep(index);
          this.fillDrawerSelects();
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
        this.paramsAdvancedOpen = false;
        this.paramFieldErrors = {};
        this.drawerSnapshot = this.captureDrawerSnapshot(type, idx);

        if (type !== 'step') {
          this.drawerStepFormReady = false;
          this.drawerStep = null;
          this.drawerStepIndex = -1;
          return;
        }

        this.drawerStepFormReady = false;
        this.drawerStep = null;
        this.drawerStepIndex = idx;
        const applyStepForm = () => {
          this.resetDrawerStep(idx);
          this.drawerStepFormReady = true;
          if (typeof this.$nextTick === 'function') {
            this.$nextTick(() => this.fillDrawerSelects());
            return;
          }
          queueMicrotask(() => this.fillDrawerSelects());
        };
        if (typeof this.$nextTick === 'function') {
          this.$nextTick(applyStepForm);
          return;
        }
        queueMicrotask(applyStepForm);
      },

      finishDrawerSession() {
        this.drawerDirty = false;
        this.drawerSnapshot = null;
        this.selectedNode = null;
        this.drawerStepFormReady = false;
        this.drawerStep = null;
        this.drawerStepIndex = -1;
        this.drawerOpen = false;
      },

      async saveDrawer() {
        if (!this.selectedNode) return;
        const { type, index } = this.selectedNode;
        if (type === 'step') {
          const paramErr = this.validateStepParams(index);
          if (paramErr) {
            showToast(paramErr, 'error');
            return;
          }
        }
        // Drawer Save persists draft work-in-progress. Publish-readiness
        // errors (auth, missing steps, etc.) still surface via validate()
        // and disable Publish — they must not block draft save.
        if (this.drawerDirty) {
          this.pushUndo();
          this.markDirty();
        }
        this.finishDrawerSession();
        this.validate();
        await this.saveDraft();
      },

      closeDrawer() {
        if (this.drawerDirty) {
          var self = this;
          showConfirmModal({
            title: flowbotI18n(
              'client.pipeline.discard_changes.title',
              'Discard Changes',
            ),
            message: flowbotI18n(
              'client.pipeline.discard_changes.message',
              'You have unsaved changes. Discard them?',
            ),
            confirmText: flowbotI18n(
              'client.pipeline.confirm_discard',
              'Discard',
            ),
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

      openVariablePicker(stepIdx, paramName) {
        this.variablePickerTarget = {
          stepIdx,
          paramName: paramName || null,
        };
        this.variablePickerOpen = true;
      },

      insertVariable(path) {
        if (!this.variablePickerTarget) {
          return;
        }
        const { stepIdx, paramName } = this.variablePickerTarget;
        const template = '{{' + path + '}}';

        if (paramName) {
          const pDef = this.getParamDef(stepIdx, paramName);
          const isFunctionsInvokeField =
            this.isFunctionsInvokeStep(stepIdx) &&
            (paramName === 'name' || paramName === 'version');
          const isNumber =
            pDef &&
            (pDef.type === 'int' ||
              pDef.type === 'int64' ||
              pDef.type === 'number');
          const isAny = pDef && pDef.type === 'any';
          let current;
          if (isFunctionsInvokeField && paramName === 'name') {
            current = this.getFunctionInvokeName(stepIdx);
          } else if (isFunctionsInvokeField && paramName === 'version') {
            current = this.getFunctionInvokeVersion(stepIdx);
          } else if (isNumber) {
            current = this.getStepParamNumber(stepIdx, paramName);
          } else if (isAny) {
            current = this.getStepParamAnyJSON(stepIdx, paramName);
          } else {
            current = this.getStepParamString(stepIdx, paramName);
          }
          const input = document.querySelector(
            '[data-param-field="' + paramName + '"]',
          );
          let next;
          if (input && typeof input.selectionStart === 'number') {
            const start = input.selectionStart;
            const end = input.selectionEnd;
            next =
              current.substring(0, start) + template + current.substring(end);
            if (isFunctionsInvokeField && paramName === 'name') {
              this.setStepParamString(stepIdx, paramName, next);
            } else if (isFunctionsInvokeField && paramName === 'version') {
              this.setFunctionInvokeVersionExpr(stepIdx, next);
            } else if (isNumber) {
              this.setStepParamNumber(stepIdx, paramName, next, pDef.type);
            } else if (isAny) {
              this.setStepParamAnyJSON(stepIdx, paramName, next);
            } else {
              this.setStepParamString(stepIdx, paramName, next);
            }
            setTimeout(() => {
              input.focus();
              input.setSelectionRange(
                start + template.length,
                start + template.length,
              );
            }, 50);
          } else {
            next = current + template;
            if (isFunctionsInvokeField && paramName === 'name') {
              this.setStepParamString(stepIdx, paramName, next);
            } else if (isFunctionsInvokeField && paramName === 'version') {
              this.setFunctionInvokeVersionExpr(stepIdx, next);
            } else if (isNumber) {
              this.setStepParamNumber(stepIdx, paramName, next, pDef.type);
            } else if (isAny) {
              this.setStepParamAnyJSON(stepIdx, paramName, next);
            } else {
              this.setStepParamString(stepIdx, paramName, next);
            }
          }
        } else {
          const step = this.steps[stepIdx];
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
            this.syncDrawerStepParamsText(stepIdx);
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
          this.syncDrawerStepParamsText(stepIdx);
        }
        this.drawerDirty = true;
        this.variablePickerOpen = false;
      },

      validate() {
        this.errors = [];
        if (this.triggers.filter((t) => t.enabled).length === 0)
          this.errors.push({
            node: { type: 'trigger', index: -1 },
            message: flowbotI18n(
              'client.pipeline.trigger_required',
              'At least one trigger must be enabled',
            ),
          });
        if (this.steps.length === 0)
          this.errors.push({
            node: { type: 'step', index: -1 },
            message: flowbotI18n(
              'client.pipeline.step_required',
              'At least one step is required',
            ),
          });
        for (let i = 0; i < this.triggers.length; i++) {
          const t = this.triggers[i];
          if (!t.enabled) continue;
          if (t.type === 'event' && !t.event)
            this.errors.push({
              node: { type: 'trigger', index: i },
              message: flowbotI18n(
                'client.pipeline.event_type_required',
                'Event type is required',
              ),
            });
          if (t.type === 'webhook' && (!t.webhook || !t.webhook.path))
            this.errors.push({
              node: { type: 'trigger', index: i },
              message: flowbotI18n(
                'client.pipeline.webhook_path_required',
                'Webhook path is required',
              ),
            });
          if (t.type === 'webhook') {
            var auth = t.webhook && t.webhook.auth ? t.webhook.auth : null;
            if (!auth || (!auth.token && !auth.hmac_secret))
              this.errors.push({
                node: { type: 'trigger', index: i },
                message: flowbotI18n(
                  'client.pipeline.auth_required',
                  'At least one auth method is required',
                ),
              });
          }
          if (t.type === 'cron' && !t.cron)
            this.errors.push({
              node: { type: 'trigger', index: i },
              message: flowbotI18n(
                'client.pipeline.cron_required',
                'Cron expression is required',
              ),
            });
        }
        for (let i = 0; i < this.steps.length; i++) {
          const s = this.steps[i];
          if (!s.name)
            this.errors.push({
              node: { type: 'step', index: i },
              message: flowbotI18n(
                'client.pipeline.step_name_required',
                'Step name is required',
              ),
            });
          if (!s.capability)
            this.errors.push({
              node: { type: 'step', index: i },
              message: flowbotI18n(
                'client.pipeline.capability_required',
                'Capability is required',
              ),
            });
          if (!s.operation)
            this.errors.push({
              node: { type: 'step', index: i },
              message: flowbotI18n(
                'client.pipeline.operation_required',
                'Operation is required',
              ),
            });
          const re = /\{\{steps\.(\w+)\./g;
          const refs = [...(s.paramsText || '').matchAll(re)].map((m) => m[1]);
          for (const ref of refs) {
            const ri = this.steps.findIndex((ss) => ss.name === ref);
            if (ri === -1)
              this.errors.push({
                node: { type: 'step', index: i },
                message: flowbotI18n(
                  'client.pipeline.upstream_var_invalid',
                  'Upstream variable steps.{{.Ref}}.* is invalid or has been removed',
                ).replace('{{.Ref}}', ref),
              });
            else if (ri >= i)
              this.errors.push({
                node: { type: 'step', index: i },
                message: flowbotI18n(
                  'client.pipeline.depends_on_above',
                  'Depends on [{{.Ref}}] which must be above this step',
                ).replace('{{.Ref}}', ref),
              });
          }
        }
        this.publishDisabled = this.errors.length > 0;
      },

      getTriggerErrorClass(idx) {
        return this.errors.some(
          (e) =>
            e.node.type === 'trigger' &&
            (e.node.index === idx || e.node.index === -1),
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

      getNodeErrorMessages(type, idx) {
        return this.errors
          .filter((e) => e.node.type === type && e.node.index === idx)
          .map((e) => e.message);
      },

      hasTriggerZoneError() {
        return this.errors.some(
          (e) => e.node.type === 'trigger' && e.node.index === -1,
        );
      },

      getTriggerZoneErrorMessage() {
        const err = this.errors.find(
          (e) => e.node.type === 'trigger' && e.node.index === -1,
        );
        return err ? err.message : '';
      },

      // webhookURL builds the absolute pipeline webhook endpoint for a path
      // (routes are served as /webhook/{path}). When token is set, append
      // ?token= so the URL is ready to call (GET-friendly; also accepted on POST).
      webhookURL(path, token) {
        if (!path) {
          return '';
        }
        var trimmed = String(path).replace(/^\/+/, '');
        if (!trimmed) {
          return '';
        }
        var url = window.location.origin + '/webhook/' + trimmed;
        if (token) {
          url += '?token=' + encodeURIComponent(token);
        }
        return url;
      },

      webhookMethod(t) {
        if (!t || !t.webhook || !t.webhook.method) {
          return 'POST';
        }
        return String(t.webhook.method).toUpperCase();
      },

      webhookToken(t) {
        if (!t || !t.webhook || !t.webhook.auth) {
          return '';
        }
        return t.webhook.auth.token || '';
      },

      webhookTriggerLabel(t) {
        if (!t || !t.webhook || !t.webhook.path) {
          return flowbotI18n('client.pipeline.webhook_preview', 'Webhook: ...');
        }
        var url = this.webhookURL(t.webhook.path, this.webhookToken(t));
        if (!url) {
          return flowbotI18n('client.pipeline.webhook_preview', 'Webhook: ...');
        }
        return this.webhookMethod(t) + ' ' + url;
      },

      webhookAuthHint(t) {
        if (!t || !t.webhook) {
          return '';
        }
        var auth = t.webhook.auth || {};
        if (auth.token) {
          return flowbotI18n(
            'client.pipeline.auth_token_preview',
            'Auth: ?token=... or header X-Webhook-Token',
          );
        }
        if (auth.hmac_secret) {
          return flowbotI18n(
            'client.pipeline.auth_hmac_preview',
            'Auth: header X-Hub-Signature-256',
          );
        }
        return flowbotI18n(
          'client.pipeline.auth_configure',
          'Auth: configure Token or HMAC Secret',
        );
      },

      webhookCurlExample(t) {
        if (!t || !t.webhook || !t.webhook.path) {
          return '';
        }
        var token = this.webhookToken(t);
        var url = this.webhookURL(t.webhook.path, token);
        if (!url) {
          return '';
        }
        var method = this.webhookMethod(t);
        var parts = ['curl', '-X', method];
        if (method === 'POST' || method === 'PUT') {
          parts.push('-H', '"Content-Type: application/json"', '-d', "'{}'");
        }
        parts.push('"' + url + '"');
        return parts.join(' ');
      },

      async copyTextValue(text, okMessage) {
        if (!text) {
          return;
        }
        try {
          if (navigator.clipboard && navigator.clipboard.writeText) {
            await navigator.clipboard.writeText(text);
          } else {
            var area = document.createElement('textarea');
            area.value = text;
            area.setAttribute('readonly', '');
            area.style.position = 'fixed';
            area.style.left = '-9999px';
            document.body.appendChild(area);
            area.select();
            try {
              if (!document.execCommand('copy')) {
                throw new Error('copy failed');
              }
            } finally {
              document.body.removeChild(area);
            }
          }
          showToast(
            okMessage || flowbotI18n('client.clip.copied', 'Copied'),
            'success',
          );
        } catch {
          showToast(
            flowbotI18n('client.pipeline.copy_failed', 'Failed to copy'),
            'error',
          );
        }
      },

      async copyWebhookURL(t) {
        if (!t || !t.webhook || !t.webhook.path) {
          showToast(
            flowbotI18n(
              'client.pipeline.webhook_path_required',
              'Webhook path is required',
            ),
            'error',
          );
          return;
        }
        var url = this.webhookURL(t.webhook.path, this.webhookToken(t));
        if (!url) {
          showToast(
            flowbotI18n(
              'client.pipeline.webhook_path_required',
              'Webhook path is required',
            ),
            'error',
          );
          return;
        }
        await this.copyTextValue(
          url,
          flowbotI18n(
            'client.pipeline.webhook_url_copied',
            'Webhook URL copied',
          ),
        );
      },

      async copyWebhookCurl(t) {
        var example = this.webhookCurlExample(t);
        if (!example) {
          showToast(
            flowbotI18n(
              'client.pipeline.webhook_path_required',
              'Webhook path is required',
            ),
            'error',
          );
          return;
        }
        await this.copyTextValue(
          example,
          flowbotI18n('client.pipeline.curl_copied', 'curl example copied'),
        );
      },

      formatErrorMessage(err) {
        const { type, index } = err.node;
        if (index < 0) {
          return err.message;
        }
        if (type === 'step') {
          const name =
            this.steps[index]?.name ||
            flowbotI18n(
              'client.pipeline.step_display_name',
              'Step {{.Index}}',
            ).replace('{{.Index}}', String(index + 1));
          return name + ': ' + err.message;
        }
        if (type === 'trigger') {
          return flowbotI18n(
            'client.pipeline.trigger_prefix',
            'Trigger {{.Index}}: {{.Error}}',
          )
            .replace('{{.Index}}', String(index + 1))
            .replace('{{.Error}}', err.message);
        }
        return err.message;
      },

      focusError(err) {
        if (!err?.node || err.node.index < 0) {
          return;
        }
        this.selectNode(err.node.type, err.node.index);
      },

      onTriggerEnabledChange() {
        this.markDirty();
        this.validate();
      },

      toggleCodeView() {
        if (this.codeView) {
          try {
            this.parseYamlToState(this.yamlText);
            this.codeView = false;
            this.validate();
          } catch (e) {
            showToast(
              flowbotI18n(
                'client.pipeline.yaml_syntax_error',
                'YAML syntax error. Fix errors before switching back to visual mode.\n{{.Error}}',
              ).replace('{{.Error}}', e.message),
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
          const resp = await fetch(this.pipelineURL(), {
            method: 'PUT',
            headers: flowbotCSRFHeaders({ 'Content-Type': 'application/json' }),
            body: JSON.stringify({ yaml, version: this.version }),
          });
          if (resp.status === 409) {
            showToast(
              flowbotI18n(
                'client.pipeline.draft_conflict',
                'This draft was modified elsewhere. Please refresh the page.',
              ),
              'error',
            );
            return;
          }
          if (!resp.ok) {
            throw new Error('HTTP ' + resp.status);
          }
          const data = await resp.json();
          this.version = data.version;
          this.status = data.status;
          this.dirty = false;
          if (this.status === 'published') {
            showToast(
              flowbotI18n(
                'client.pipeline.draft_saved_publish_hint',
                'Draft saved. Click Publish to update the live webhook.',
              ),
              'success',
            );
          } else {
            showToast(
              flowbotI18n('client.pipeline.draft_saved', 'Draft saved'),
              'success',
            );
          }
        } catch (e) {
          console.error('Auto-save failed:', e);
          showToast(
            flowbotI18n(
              'client.pipeline.save_failed',
              'Save failed. Your changes are not saved yet.',
            ),
            'error',
          );
        } finally {
          this.saving = false;
        }
      },

      async publish() {
        if (this.publishDisabled) return;
        this.publishing = true;
        await this.saveDraft();
        try {
          const resp = await fetch(this.pipelineURL('/publish'), {
            method: 'PUT',
            headers: flowbotCSRFHeaders({ 'Content-Type': 'application/json' }),
            body: JSON.stringify({ version: this.version }),
          });
          if (resp.status === 409) {
            showToast(
              flowbotI18n(
                'client.pipeline.draft_conflict',
                'This draft was modified elsewhere. Please refresh the page.',
              ),
              'error',
            );
            return;
          }
          if (!resp.ok) {
            throw new Error('HTTP ' + resp.status);
          }
          const data = await resp.json();
          this.version = data.version;
          this.status = 'published';
          showToast(
            flowbotI18n('client.pipeline.published', 'Pipeline published'),
            'success',
          );
        } catch (e) {
          console.error('Publish failed:', e);
          showToast(
            flowbotI18n(
              'client.pipeline.publish_failed',
              'Publish failed: {{.Error}}',
            ).replace('{{.Error}}', e.message),
            'error',
          );
        } finally {
          this.publishing = false;
        }
      },

      async loadMockPayload() {
        try {
          const resp = await fetch(
            this.pipelineURL('/mock?source=' + this.testTriggerSource),
          );
          if (!resp.ok) {
            throw new Error('HTTP ' + resp.status);
          }
          const data = await resp.json();
          this.testMockPayload = JSON.stringify(data.payload, null, 2);
        } catch (e) {
          console.error('Failed to load mock payload:', e);
          showToast(
            flowbotI18n(
              'client.pipeline.mock_payload_failed',
              'Failed to load mock payload',
            ),
            'error',
          );
        }
      },

      async runTest() {
        await this.saveDraft();
        const upToIdx = this.selectedNode?.index;
        if (upToIdx === null || upToIdx === undefined) return;
        this.testing = true;
        try {
          const resp = await fetch(this.pipelineURL('/test'), {
            method: 'POST',
            headers: flowbotCSRFHeaders({ 'Content-Type': 'application/json' }),
            body: JSON.stringify({
              trigger_source: this.testTriggerSource,
              mock_payload: JSON.parse(this.testMockPayload || '{}'),
              up_to_step_index: upToIdx,
            }),
          });
          this.testResults = await resp.json();
        } catch (e) {
          console.error('Test failed:', e);
          this.testResults = { success: false, error: e.message };
          showToast(
            flowbotI18n(
              'client.pipeline.test_failed',
              'Test failed: {{.Error}}',
            ).replace('{{.Error}}', e.message),
            'error',
          );
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
            flowbotI18n(
              'client.pipeline.move_blocked',
              'Cannot move: this step depends on data from a step at or above the target position.',
            ),
            'warning',
          );
          return;
        }

        this.pushUndo();
        var item = this.steps.splice(this.dragFromIdx, 1)[0];
        this.steps.splice(idx, 0, item);
        this.markDirty();
        this.validate();
        this.validateSelectedNode();
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
        const input =
          this.$el.querySelector('#yaml-import-input') ||
          document.getElementById('yaml-import-input');
        if (input) {
          input.click();
        }
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
            showToast(
              flowbotI18n(
                'client.pipeline.invalid_yaml',
                'Invalid YAML: not a pipeline definition',
              ),
              'error',
            );
            return;
          }
          this.pushUndo();
          this.parseYamlToState(text);
          this.markDirty();
          this.validate();
          showToast(
            flowbotI18n(
              'client.pipeline.yaml_import_success',
              'YAML imported successfully',
            ),
            'success',
          );
        } catch (err) {
          showToast(
            flowbotI18n(
              'client.pipeline.yaml_import_failed',
              'Import failed: {{.Error}}',
            ).replace('{{.Error}}', err.message),
            'error',
          );
        } finally {
          e.target.value = '';
        }
      },

      async loadVersions() {
        this.historyLoading = true;
        try {
          var resp = await fetch(this.pipelineURL('/versions'));
          if (!resp.ok) {
            this.versions = [];
            showToast(
              flowbotI18n(
                'client.pipeline.versions_load_failed',
                'Failed to load versions',
              ),
              'error',
            );
            return;
          }
          this.versions = await resp.json();
        } catch (e) {
          console.error('Failed to load versions:', e);
          this.versions = [];
          showToast(
            flowbotI18n(
              'client.pipeline.versions_load_failed',
              'Failed to load versions',
            ),
            'error',
          );
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
          var resp = await fetch(this.pipelineURL('/versions/' + v.version));
          if (!resp.ok) throw new Error('Not found');
          var data = await resp.json();
          this.selectedVersionYaml = data.yaml;
        } catch (e) {
          console.error('Failed to load version:', e);
          this.selectedVersionYaml = '';
          showToast(
            flowbotI18n(
              'client.pipeline.version_load_failed',
              'Failed to load version',
            ),
            'error',
          );
        } finally {
          this.historyLoading = false;
        }
      },

      relativeTime(isoStr) {
        if (!isoStr) return '';
        var d = new Date(isoStr);
        if (isNaN(d.getTime())) return '';
        var now = new Date();
        var diff = now - d;
        var mins = Math.floor(diff / 60000);
        if (mins < 60) {
          return flowbotI18n(
            'client.pipeline.relative_minutes',
            '{{.Count}} minutes ago',
          ).replace('{{.Count}}', String(mins));
        }
        var hours = Math.floor(mins / 60);
        if (hours < 24) {
          return flowbotI18n(
            'client.pipeline.relative_hours',
            '{{.Count}} hours ago',
          ).replace('{{.Count}}', String(hours));
        }
        var days = Math.floor(hours / 24);
        return flowbotI18n(
          'client.pipeline.relative_days',
          '{{.Count}} days ago',
        ).replace('{{.Count}}', String(days));
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
          var resp = await fetch(self.pipelineURL('/versions/' + v.version));
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
          showToast(
            flowbotI18n(
              'client.pipeline.diff_failed',
              'Failed to compare versions',
            ),
            'error',
          );
        }
      },

      async openMemoryModal() {
        this.memoryModalOpen = true;
        this.memoryError = '';
        await this.loadMemoryFacts();
      },

      closeMemoryModal() {
        this.memoryModalOpen = false;
        this.memoryError = '';
      },

      async loadMemoryFacts() {
        try {
          const resp = await fetch(
            '/service/web/agent-memory/facts?scope=' +
              encodeURIComponent(this.name),
          );
          if (!resp.ok) {
            throw new Error('failed to list memory facts');
          }
          const json = await resp.json();
          const facts = json.data || [];
          this.memoryKeys = facts.map((f) => f.key);
          if (this.memoryKeys.length === 0) {
            this.memorySelectedKey = '';
            this.memoryContent = '';
            this.memoryPinned = false;
            return;
          }
          if (!this.memoryKeys.includes(this.memorySelectedKey)) {
            this.memorySelectedKey = this.memoryKeys[0];
          }
          await this.loadMemoryFact();
        } catch (e) {
          console.error('Failed to load memory facts:', e);
          this.memoryError = flowbotI18n(
            'client.pipeline.memory_load_failed',
            'Failed to load memory facts',
          );
          this.memoryKeys = [];
        }
      },

      async loadMemoryFact() {
        if (!this.name || !this.memorySelectedKey) {
          return;
        }
        try {
          const resp = await fetch(
            '/service/web/agent-memory/facts?scope=' +
              encodeURIComponent(this.name),
          );
          if (!resp.ok) {
            throw new Error('failed to load memory facts');
          }
          const json = await resp.json();
          const facts = json.data || [];
          const fact = facts.find((f) => f.key === this.memorySelectedKey);
          this.memoryContent = (fact && fact.value) || '';
          this.memoryPinned = !!(fact && fact.pinned);
          this.memoryError = '';
        } catch (e) {
          console.error('Failed to load memory fact:', e);
          this.memoryError = flowbotI18n(
            'client.pipeline.memory_fact_load_failed',
            'Failed to load memory fact',
          );
        }
      },

      async saveMemoryFact() {
        if (!this.name) {
          return;
        }
        const key = (this.memorySelectedKey || '').trim();
        if (!key) {
          this.memoryError = flowbotI18n(
            'client.pipeline.memory_key_required',
            'Key is required',
          );
          return;
        }
        this.memorySaving = true;
        this.memoryError = '';
        try {
          const resp = await fetch('/service/web/agent-memory/facts', {
            method: 'PUT',
            headers: flowbotCSRFHeaders({ 'Content-Type': 'application/json' }),
            body: JSON.stringify({
              scope: this.name,
              key: key,
              value: this.memoryContent,
              pinned: !!this.memoryPinned,
            }),
          });
          if (!resp.ok) {
            throw new Error('failed to save memory fact');
          }
          this.closeMemoryModal();
        } catch (e) {
          console.error('Failed to save memory fact:', e);
          this.memoryError = flowbotI18n(
            'client.pipeline.memory_fact_save_failed',
            'Failed to save memory fact',
          );
        } finally {
          this.memorySaving = false;
        }
      },

      async deleteMemoryFact() {
        if (!this.name || !this.memorySelectedKey) {
          return;
        }
        this.memorySaving = true;
        this.memoryError = '';
        try {
          const resp = await fetch(
            '/service/web/agent-memory/facts?scope=' +
              encodeURIComponent(this.name) +
              '&key=' +
              encodeURIComponent(this.memorySelectedKey),
            {
              method: 'DELETE',
              headers: flowbotCSRFHeaders(),
            },
          );
          if (!resp.ok) {
            throw new Error('failed to delete memory fact');
          }
          this.memorySelectedKey = '';
          await this.loadMemoryFacts();
        } catch (e) {
          console.error('Failed to delete memory fact:', e);
          this.memoryError = flowbotI18n(
            'client.pipeline.memory_fact_delete_failed',
            'Failed to delete memory fact',
          );
        } finally {
          this.memorySaving = false;
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
