(function () {
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
      memoryModalOpen: false,
      memoryFiles: [],
      memorySelectedFile: 'MEMORIES.md',
      memoryContent: '',
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
          showToast('Failed to load pipeline', 'error');
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
          showToast('Pipeline name is required', 'error');
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
              (data.error && data.error.message) || 'Failed to rename pipeline';
            showToast(message, 'error');
            this.renaming = false;
            this.renameValue = this.name;
            return;
          }
          const renamed = (data && data.name) || nextName;
          showToast('Pipeline renamed', 'success');
          window.location.href =
            '/service/web/pipelines/' + encodeURIComponent(renamed);
        } catch (e) {
          console.error('Failed to rename pipeline:', e);
          showToast('Failed to rename pipeline', 'error');
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
          showToast('Failed to load agent run options', 'error');
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
        } catch (e) {
          console.error('Failed to load capabilities:', e);
          showToast('Failed to load capabilities', 'error');
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

      hasTestResultSteps() {
        return !!(this.testResults && this.testResults.steps);
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

      getFormOperationInput(idx) {
        if (idx == null) return [];
        return this.getCurrentOperationInput(idx).filter((p) => {
          if (this.isAgentRunStringListParam(idx, p.name)) {
            return false;
          }
          return true;
        });
      },

      isParamTypeString(p) {
        return p?.type === 'string';
      },

      isParamTypeNumber(p) {
        return p?.type === 'int' || p?.type === 'int64';
      },

      isParamTypeBool(p) {
        return p?.type === 'bool';
      },

      isParamTypeStringList(p) {
        return p?.type === '[]string';
      },

      isParamTypeMap(p) {
        return p?.type === 'map[string]any';
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
        return p && p.required ? '0 or {{expr}}' : 'optional';
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
            return Number(value) === Number(def);
          case 'bool':
            return value === def;
          case '[]string':
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
          default:
            return false;
        }
      },

      isAgentRunStep(idx) {
        const step = this.steps[idx];
        return step?.capability === 'agent' && step?.operation === 'run';
      },

      pipelineMemoryEnabled() {
        if (!this.name) {
          return false;
        }
        const available = (this.agentRunOptions.tools || []).includes(
          'update_memory',
        );
        if (!available) {
          return false;
        }
        for (let i = 0; i < this.steps.length; i++) {
          if (!this.isAgentRunStep(i)) {
            continue;
          }
          if (this.getAgentRunParamList(i, 'tools').includes('update_memory')) {
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
            return value === '' || Number.isNaN(Number(value));
          case 'bool':
            return value === 'unset';
          case '[]string':
            return !Array.isArray(value) || value.length === 0;
          case 'map[string]any':
            return (
              typeof value !== 'object' ||
              value === null ||
              Array.isArray(value) ||
              Object.keys(value).length === 0
            );
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
            return val === '' || Number.isNaN(Number(val));
          case 'bool':
            return false;
          case '[]string':
            return !Array.isArray(val) || val.length === 0;
          case 'map[string]any':
            return (
              typeof val !== 'object' ||
              val === null ||
              Array.isArray(val) ||
              Object.keys(val).length === 0
            );
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
          return 'Invalid params JSON: ' + e.message;
        }
        const input = this.getCurrentOperationInput(idx);
        const params = this.parseStepParams(idx);
        for (const p of input) {
          if (p.required && this.isParamValueMissing(params[p.name], p.type)) {
            return 'Parameter "' + p.name + '" is required';
          }
          if (
            p.type === 'map[string]any' &&
            this.isParamFieldError(idx, p.name)
          ) {
            return 'Parameter "' + p.name + '" has invalid JSON';
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
          webhook: {
            path: '',
            method: 'POST',
            auth: { token: '', hmac_secret: '' },
          },
        });
        this.markDirty();
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
        this.syncDrawerAfterListRemoval('step', idx);
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
        this.paramsAdvancedOpen = false;
        this.paramFieldErrors = {};
        this.drawerSnapshot = this.captureDrawerSnapshot(type, idx);
      },

      finishDrawerSession() {
        this.drawerDirty = false;
        this.drawerSnapshot = null;
        this.selectedNode = null;
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
          const isNumber =
            pDef && (pDef.type === 'int' || pDef.type === 'int64');
          const current = isNumber
            ? this.getStepParamNumber(stepIdx, paramName)
            : this.getStepParamString(stepIdx, paramName);
          const input = document.querySelector(
            '[data-param-field="' + paramName + '"]',
          );
          let next;
          if (input && typeof input.selectionStart === 'number') {
            const start = input.selectionStart;
            const end = input.selectionEnd;
            next =
              current.substring(0, start) + template + current.substring(end);
            if (isNumber) {
              this.setStepParamNumber(stepIdx, paramName, next, pDef.type);
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
            if (isNumber) {
              this.setStepParamNumber(stepIdx, paramName, next, pDef.type);
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

      formatErrorMessage(err) {
        const { type, index } = err.node;
        if (index < 0) {
          return err.message;
        }
        if (type === 'step') {
          const name = this.steps[index]?.name || 'Step ' + (index + 1);
          return name + ': ' + err.message;
        }
        if (type === 'trigger') {
          return 'Trigger ' + (index + 1) + ': ' + err.message;
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
          const resp = await fetch(this.pipelineURL(), {
            method: 'PUT',
            headers: flowbotCSRFHeaders({ 'Content-Type': 'application/json' }),
            body: JSON.stringify({ yaml, version: this.version }),
          });
          if (resp.status === 409) {
            showToast(
              'This draft was modified elsewhere. Please refresh the page.',
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
          showToast('Draft saved', 'success');
        } catch (e) {
          console.error('Auto-save failed:', e);
          showToast('Save failed. Your changes are not saved yet.', 'error');
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
              'This draft was modified elsewhere. Please refresh the page.',
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
            this.pipelineURL('/mock?source=' + this.testTriggerSource),
          );
          if (!resp.ok) {
            throw new Error('HTTP ' + resp.status);
          }
          const data = await resp.json();
          this.testMockPayload = JSON.stringify(data.payload, null, 2);
        } catch (e) {
          console.error('Failed to load mock payload:', e);
          showToast('Failed to load mock payload', 'error');
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
          var resp = await fetch(this.pipelineURL('/versions'));
          if (!resp.ok) {
            this.versions = [];
            showToast('Failed to load versions', 'error');
            return;
          }
          this.versions = await resp.json();
        } catch (e) {
          console.error('Failed to load versions:', e);
          this.versions = [];
          showToast('Failed to load versions', 'error');
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
          showToast('Failed to load version', 'error');
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
          showToast('Failed to compare versions', 'error');
        }
      },

      async openMemoryModal() {
        this.memoryModalOpen = true;
        this.memoryError = '';
        await this.loadMemoryFiles();
        await this.loadMemoryContent();
      },

      closeMemoryModal() {
        this.memoryModalOpen = false;
        this.memoryError = '';
      },

      async loadMemoryFiles() {
        try {
          const resp = await fetch(
            '/service/web/agent-memory/files?scope=' +
              encodeURIComponent(this.name),
          );
          if (!resp.ok) {
            throw new Error('failed to list memory files');
          }
          const json = await resp.json();
          const files = json.data || [];
          this.memoryFiles = files.length > 0 ? files : ['MEMORIES.md'];
          if (!this.memoryFiles.includes(this.memorySelectedFile)) {
            this.memorySelectedFile = this.memoryFiles[0];
          }
        } catch (e) {
          console.error('Failed to load memory files:', e);
          this.memoryError = 'Failed to load memory files';
          this.memoryFiles = ['MEMORIES.md'];
        }
      },

      async loadMemoryContent() {
        if (!this.name) {
          return;
        }
        try {
          const url =
            '/service/web/agent-memory/content?scope=' +
            encodeURIComponent(this.name) +
            '&file=' +
            encodeURIComponent(this.memorySelectedFile || 'MEMORIES.md');
          const resp = await fetch(url);
          if (!resp.ok) {
            throw new Error('failed to load memory content');
          }
          const json = await resp.json();
          this.memoryContent = (json.data && json.data.content) || '';
          this.memoryError = '';
        } catch (e) {
          console.error('Failed to load memory content:', e);
          this.memoryError = 'Failed to load memory content';
        }
      },

      async saveMemoryContent() {
        if (!this.name) {
          return;
        }
        this.memorySaving = true;
        this.memoryError = '';
        try {
          const resp = await fetch('/service/web/agent-memory/content', {
            method: 'PUT',
            headers: flowbotCSRFHeaders({ 'Content-Type': 'application/json' }),
            body: JSON.stringify({
              scope: this.name,
              file: this.memorySelectedFile || 'MEMORIES.md',
              content: this.memoryContent,
            }),
          });
          if (!resp.ok) {
            throw new Error('failed to save memory content');
          }
          this.closeMemoryModal();
        } catch (e) {
          console.error('Failed to save memory content:', e);
          this.memoryError = 'Failed to save memory content';
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
