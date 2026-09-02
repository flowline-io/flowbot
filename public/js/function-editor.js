(function () {
  function toastCopied() {
    window.dispatchEvent(
      new CustomEvent('flowbot:toast', {
        detail: {
          type: 'success',
          message: flowbotI18n(
            'client.function_editor.copied_call_url',
            'Copied call URL',
          ),
        },
      }),
    );
  }

  function register() {
    Alpine.data('functionEditor', () => ({
      name: '',
      version: 1,
      status: 'draft',
      entrypoint: 'main.py',
      source: '',
      envText: '',
      token: '',
      hmacSecret: '',
      tokenSet: false,
      hmacSet: false,
      publishedVersion: null,
      hasUnpublishedChanges: false,
      callURL: '',
      dirty: false,
      tab: 'code',
      tryEvent: '{}',
      tryResult: '',
      busy: false,

      init() {
        var root = this.$el;
        this.name = root.getAttribute('data-function-name') || '';
        this.version =
          parseInt(root.getAttribute('data-version') || '1', 10) || 1;
        this.status = root.getAttribute('data-status') || 'draft';
        this.entrypoint = root.getAttribute('data-entrypoint') || 'main.py';
        this.tokenSet = root.getAttribute('data-token-set') === 'true';
        this.hmacSet = root.getAttribute('data-hmac-set') === 'true';
        this.hasUnpublishedChanges =
          root.getAttribute('data-has-unpublished') === 'true';
        var pub = root.getAttribute('data-published-version') || '';
        this.publishedVersion = pub ? parseInt(pub, 10) : null;
        this.callURL = root.getAttribute('data-call-url') || '';
        if (!this.callURL) {
          var callEl = root.querySelector('[data-testid="function-call-url"]');
          if (callEl) {
            this.callURL = String(callEl.textContent || '').trim();
          }
        }
        var sourceEl = root.querySelector('[data-testid="input-source"]');
        if (sourceEl) {
          this.source = sourceEl.value;
        }
        var envEl = root.querySelector('[data-testid="input-env"]');
        if (envEl) {
          this.envText = envEl.value;
        }
        var tokenEl = root.querySelector('[data-testid="input-token"]');
        if (tokenEl && this.tokenSet) {
          this.token = tokenEl.getAttribute('placeholder') || '••••••••';
        }
        var hmacEl = root.querySelector('[data-testid="input-hmac"]');
        if (hmacEl && this.hmacSet) {
          this.hmacSecret = hmacEl.getAttribute('placeholder') || '••••••••';
        }
      },

      callVersionURL() {
        if (!this.publishedVersion) {
          return '';
        }
        var base = String(this.callURL || '')
          .trim()
          .replace(/\/$/, '');
        if (!base) {
          return '';
        }
        return base + '/v/' + this.publishedVersion;
      },

      baseURL() {
        return '/service/web/functions/' + encodeURIComponent(this.name);
      },

      copyCallLink(event) {
        var el = event && event.currentTarget;
        this.copyCallURL(el ? el.getAttribute('data-copy-url') : '');
      },

      copyCallURL(url) {
        var text = String(url || '');
        if (!text) {
          return;
        }
        if (navigator.clipboard && navigator.clipboard.writeText) {
          navigator.clipboard
            .writeText(text)
            .then(toastCopied)
            .catch(function () {});
          return;
        }
        var area = document.createElement('textarea');
        area.value = text;
        area.setAttribute('readonly', '');
        area.style.position = 'fixed';
        area.style.left = '-9999px';
        document.body.appendChild(area);
        area.select();
        try {
          document.execCommand('copy');
          toastCopied();
        } finally {
          document.body.removeChild(area);
        }
      },

      setTab(name) {
        this.tab = name;
      },

      markDirty() {
        this.dirty = true;
      },

      buildMetadataYAML() {
        var lines = ['name: ' + this.name, 'http:', '  auth:'];
        lines.push('    token: ' + JSON.stringify(this.token || ''));
        if (this.hmacSecret) {
          lines.push('    hmac_secret: ' + JSON.stringify(this.hmacSecret));
        }
        var envLines = (this.envText || '').split(/\r?\n/);
        var envPairs = [];
        for (var i = 0; i < envLines.length; i++) {
          var line = envLines[i].trim();
          if (!line) continue;
          var eq = line.indexOf('=');
          if (eq <= 0) continue;
          envPairs.push({
            key: line.slice(0, eq).trim(),
            value: line.slice(eq + 1),
          });
        }
        if (envPairs.length) {
          lines.push('env:');
          for (var j = 0; j < envPairs.length; j++) {
            lines.push(
              '  ' + envPairs[j].key + ': ' + JSON.stringify(envPairs[j].value),
            );
          }
        }
        return lines.join('\n') + '\n';
      },

      async saveDraft() {
        if (this.busy) return;
        this.busy = true;
        try {
          var resp = await fetch(this.baseURL(), {
            method: 'PUT',
            headers: flowbotCSRFHeaders({
              'Content-Type': 'application/json',
              Accept: 'application/json',
            }),
            credentials: 'same-origin',
            body: JSON.stringify({
              metadata: this.buildMetadataYAML(),
              entrypoint: this.entrypoint,
              source: this.source,
              version: this.version,
            }),
          });
          var data = await resp.json().catch(function () {
            return {};
          });
          if (!resp.ok) {
            var msg =
              (data.error && data.error.message) ||
              data.message ||
              flowbotI18n(
                'client.function_editor.save_draft_failed',
                'Failed to save draft',
              );
            window.dispatchEvent(
              new CustomEvent('flowbot:toast', {
                detail: { type: 'error', message: msg },
              }),
            );
            if (resp.status === 409) {
              window.location.reload();
            }
            return;
          }
          this.version = data.version || this.version;
          this.status = data.status || this.status;
          this.hasUnpublishedChanges = !!data.has_unpublished_changes;
          if (data.published_version) {
            this.publishedVersion = data.published_version;
          }
          this.dirty = false;
          window.dispatchEvent(
            new CustomEvent('flowbot:toast', {
              detail: {
                type: 'success',
                message: flowbotI18n(
                  'client.function_editor.draft_saved',
                  'Draft saved',
                ),
              },
            }),
          );
        } finally {
          this.busy = false;
        }
      },

      async publish() {
        if (this.busy) return;
        if (this.dirty) {
          await this.saveDraft();
          if (this.dirty) return;
        }
        this.busy = true;
        try {
          var resp = await fetch(this.baseURL() + '/publish', {
            method: 'POST',
            headers: flowbotCSRFHeaders({
              'Content-Type': 'application/json',
              Accept: 'application/json',
            }),
            credentials: 'same-origin',
            body: JSON.stringify({ version: this.version }),
          });
          var data = await resp.json().catch(function () {
            return {};
          });
          if (!resp.ok) {
            var msg =
              (data.error && data.error.message) ||
              data.message ||
              flowbotI18n(
                'client.function_editor.publish_failed',
                'Publish failed: {{.Error}}',
              ).replace('{{.Error}}', '');
            window.dispatchEvent(
              new CustomEvent('flowbot:toast', {
                detail: { type: 'error', message: msg },
              }),
            );
            if (resp.status === 409) {
              window.location.reload();
            }
            return;
          }
          this.version = data.definition_version || this.version;
          this.status = data.status || 'published';
          this.publishedVersion = data.version || this.publishedVersion;
          this.hasUnpublishedChanges = false;
          this.dirty = false;
          window.dispatchEvent(
            new CustomEvent('flowbot:toast', {
              detail: {
                type: 'success',
                message: flowbotI18n(
                  'client.function_editor.publish_success',
                  'Function published',
                ),
              },
            }),
          );
        } finally {
          this.busy = false;
        }
      },

      async tryInvoke() {
        if (this.busy || !this.publishedVersion) return;
        this.busy = true;
        this.tryResult = flowbotI18n(
          'client.function_editor.running',
          'Running…',
        );
        try {
          var eventPayload = {};
          try {
            eventPayload = JSON.parse(this.tryEvent || '{}');
          } catch {
            this.tryResult = flowbotI18n(
              'client.function_editor.invalid_event_json',
              'Invalid event JSON',
            );
            return;
          }
          var resp = await fetch(this.baseURL() + '/try', {
            method: 'POST',
            headers: flowbotCSRFHeaders({
              'Content-Type': 'application/json',
              Accept: 'application/json',
            }),
            credentials: 'same-origin',
            body: JSON.stringify({ event: eventPayload }),
          });
          var data = await resp.json().catch(function () {
            return {};
          });
          if (!resp.ok) {
            this.tryResult =
              (data.error && data.error.message) ||
              data.message ||
              flowbotI18n(
                'client.function_editor.invoke_failed',
                'Invoke failed',
              );
            return;
          }
          this.tryResult = JSON.stringify(data, null, 2);
        } finally {
          this.busy = false;
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
