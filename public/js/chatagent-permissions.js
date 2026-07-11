// chatagent-permissions.js wires dynamic pattern rows for the permissions form.
(function () {
  function isBroadPattern(pattern) {
    const trimmed = (pattern || '').trim();
    return trimmed === '' || trimmed === '*' || trimmed === '**';
  }

  function buildPatternRow(key, idx) {
    return (
      '<div class="pattern-row flex flex-wrap items-start gap-2">' +
      '<input type="text" name="perm[' +
      key +
      '][patterns][' +
      idx +
      '][pattern]" data-testid="perm-' +
      key +
      '-pattern-' +
      idx +
      '" class="input input-bordered input-sm font-mono flex-1 min-w-[12rem]" placeholder="git *" />' +
      '<select name="perm[' +
      key +
      '][patterns][' +
      idx +
      '][action]" data-testid="perm-' +
      key +
      '-action-' +
      idx +
      '" class="select select-bordered select-sm w-28">' +
      '<option value="ask" selected>ask</option>' +
      '<option value="allow">allow</option>' +
      '<option value="deny">deny</option>' +
      '</select>' +
      '<button type="button" class="btn btn-ghost btn-sm pattern-row-remove" data-testid="perm-' +
      key +
      '-remove-' +
      idx +
      '">Remove</button>' +
      '</div>'
    );
  }

  function addPatternRow(container) {
    const key = container.dataset.permKey;
    const idx = Number(container.dataset.nextIndex || '0');
    container.insertAdjacentHTML('beforeend', buildPatternRow(key, idx));
    container.dataset.nextIndex = String(idx + 1);
  }

  document.addEventListener('click', function (event) {
    const addBtn = event.target.closest('.pattern-row-add');
    if (addBtn) {
      const key = addBtn.dataset.permKey;
      const container = document.getElementById('perm-patterns-' + key);
      if (container) {
        addPatternRow(container);
      }
      return;
    }
    const removeBtn = event.target.closest('.pattern-row-remove');
    if (removeBtn) {
      const row = removeBtn.closest('.pattern-row');
      if (row) {
        row.remove();
      }
    }
  });

  document.addEventListener(
    'submit',
    function (event) {
      const form = event.target;
      if (
        !form ||
        form.getAttribute('data-testid') !== 'chatagent-permissions-form'
      ) {
        return;
      }
      const mode = document.getElementById('chatagent-permissions-submit-mode');
      if (!mode || mode.value !== 'form') {
        return;
      }
      const inputs = form.querySelectorAll('.pattern-row input[type="text"]');
      for (const input of inputs) {
        if (isBroadPattern(input.value)) {
          event.preventDefault();
          input.classList.add('border-red-500');
          input.focus();
          return;
        }
        input.classList.remove('border-red-500');
      }
    },
    true,
  );
})();
