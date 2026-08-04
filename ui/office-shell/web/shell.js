/* Shared Office chrome helpers (Collab v2 / Wave F / O-SHELL). */
(function (global) {
  function markActiveNav(moduleId) {
    document.querySelectorAll('.era-nav a[data-mod]').forEach(function (a) {
      a.classList.toggle('active', a.getAttribute('data-mod') === moduleId);
    });
  }

  function setStatus(el, text, kind) {
    if (!el) return;
    el.textContent = text || '';
    el.className = 'era-status' + (kind ? ' ' + kind : '');
  }

  /** Save pill: idle | dirty | saving | ok | err */
  function setSavePill(el, state, detail) {
    if (!el) return;
    var map = {
      idle: { text: 'All changes saved', cls: 'ok' },
      dirty: { text: 'Unsaved changes', cls: 'dirty' },
      saving: { text: 'Saving…', cls: '' },
      ok: { text: 'Saved', cls: 'ok' },
      err: { text: detail || 'Save failed', cls: 'err' },
    };
    var m = map[state] || map.idle;
    if (state === 'ok' && detail) m.text = detail;
    el.textContent = m.text;
    el.className = 'era-save-pill' + (m.cls ? ' ' + m.cls : '');
    el.setAttribute('data-state', state || 'idle');
  }

  function wireDocTitle(input, onCommit) {
    if (!input) return;
    input.addEventListener('keydown', function (e) {
      if (e.key === 'Enter') {
        e.preventDefault();
        input.blur();
      }
    });
    input.addEventListener('blur', function () {
      var v = (input.value || '').trim();
      if (!v) {
        input.value = input.getAttribute('data-fallback') || 'Untitled';
        v = input.value;
      }
      if (typeof onCommit === 'function') onCommit(v);
    });
  }

  function mountIcons(root) {
    if (global.EraOfficeIcons && global.EraOfficeIcons.mount) {
      global.EraOfficeIcons.mount(root || document);
    }
  }

  function mountNav(root) {
    if (global.EraOfficeIcons && global.EraOfficeIcons.mountNav) {
      global.EraOfficeIcons.mountNav(root || document);
    } else {
      mountIcons(root);
    }
  }

  function loginUrl(next) {
    var n = next || location.pathname + location.search || '/drive/';
    return '/login?next=' + encodeURIComponent(n);
  }

  function parseJwtPayload(token) {
    if (!token) return null;
    try {
      var part = token.split('.')[1];
      if (!part) return null;
      return JSON.parse(atob(part.replace(/-/g, '+').replace(/_/g, '/')));
    } catch (_) {
      return null;
    }
  }

  /** True if token missing or exp is in the past (30s skew). */
  function isTokenExpired(token) {
    var t = token || localStorage.getItem('era_token') || '';
    if (!t) return true;
    var p = parseJwtPayload(t);
    if (!p) return true;
    if (p.exp == null) return false;
    return Number(p.exp) * 1000 <= Date.now() - 30 * 1000;
  }

  function redirectToLogin(next) {
    try {
      localStorage.removeItem('era_token');
    } catch (_) {}
    location.href = loginUrl(next);
  }

  function requireAuthOrRedirect() {
    var token = localStorage.getItem('era_token') || '';
    if (!token || isTokenExpired(token)) {
      redirectToLogin();
      return false;
    }
    return true;
  }

  /**
   * If Response is 401 (lost/expired token), redirect to login.
   * Does not treat 403 (ACL deny) as auth loss.
   */
  function handleUnauthorized(res) {
    if (!res || res.status !== 401) return false;
    redirectToLogin();
    return true;
  }

  /** fetch wrapper: Authorization + 401 → login. */
  function authFetch(url, opts) {
    opts = opts || {};
    var headers = Object.assign({}, opts.headers || {});
    var token = localStorage.getItem('era_token') || '';
    if (token && !headers.Authorization && !headers.authorization) {
      headers.Authorization = 'Bearer ' + token;
    }
    return fetch(url, Object.assign({}, opts, { headers: headers })).then(function (res) {
      handleUnauthorized(res);
      return res;
    });
  }

  function signOut(next) {
    localStorage.removeItem('era_token');
    location.href = loginUrl(next || '/drive/');
  }

  function syncUserChip() {
    var chip = document.getElementById('userChip');
    if (!chip) return;
    var lamp = chip.querySelector('.era-user-lamp');
    var label = chip.querySelector('.era-user-label');
    if (!lamp || !label) {
      chip.textContent = '';
      lamp = document.createElement('span');
      lamp.className = 'era-user-lamp';
      lamp.setAttribute('aria-hidden', 'true');
      label = document.createElement('span');
      label.className = 'era-user-label';
      chip.appendChild(lamp);
      chip.appendChild(label);
    }
    var token = localStorage.getItem('era_token') || '';
    if (!token || isTokenExpired(token)) {
      lamp.className = 'era-user-lamp off';
      label.textContent = 'Sign in';
      chip.title = 'Sign in to ERA Office';
      chip.style.cursor = 'pointer';
      chip.onclick = function () {
        location.href = loginUrl();
      };
      return;
    }
    lamp.className = 'era-user-lamp on';
    var p = parseJwtPayload(token) || {};
    label.textContent = p.email || p.sub || 'Account';
    chip.title = (p.email || p.sub || '') + ' — signed in · click to sign out';
    chip.style.cursor = 'pointer';
    chip.onclick = function () {
      confirmAction({
        title: 'Sign out',
        message: 'Sign out of ERA Office?',
        okLabel: 'Sign out',
        danger: true,
      }).then(function (ok) {
        if (ok) signOut();
      });
    };
  }

  function toastStatus(el, msg, isErr) {
    if (!el) return;
    var text = msg || '';
    if (!text || /^token present$/i.test(text.trim())) {
      el.classList.remove('show');
      el.textContent = '';
      return;
    }
    el.textContent = text;
    el.className = 'era-status ' + (isErr ? 'err' : 'ok') + ' show';
    clearTimeout(el._eraToastTimer);
    el._eraToastTimer = setTimeout(function () {
      el.classList.remove('show');
    }, isErr ? 5000 : 2200);
  }

  function setCommentsOpen(open) {
    var panel = document.getElementById('commentsPanel');
    var btn = document.getElementById('commentsToggleBtn');
    if (panel) {
      panel.classList.toggle('era-comments-open', !!open);
      panel.hidden = !open;
    }
    if (btn) {
      btn.classList.toggle('active', !!open);
      btn.setAttribute('aria-pressed', open ? 'true' : 'false');
    }
  }

  function wireCommentsToggle(defaultOpen) {
    var btn = document.getElementById('commentsToggleBtn');
    var panel = document.getElementById('commentsPanel');
    if (!btn || !panel) return;
    setCommentsOpen(!!defaultOpen);
    if (!btn._eraCommentsWired) {
      btn._eraCommentsWired = true;
      btn.addEventListener('click', function () {
        var open = !panel.classList.contains('era-comments-open');
        setCommentsOpen(open);
      });
    }
    var closeBtn = document.getElementById('commentsCloseBtn');
    if (closeBtn && !closeBtn._eraCommentsWired) {
      closeBtn._eraCommentsWired = true;
      closeBtn.addEventListener('click', function () {
        setCommentsOpen(false);
      });
    }
  }

  function wireSessionWatch() {
    if (document._eraSessionWatch) return;
    document._eraSessionWatch = true;
    window.addEventListener('focus', function () {
      var token = localStorage.getItem('era_token') || '';
      if (token && isTokenExpired(token)) redirectToLogin();
    });
    setInterval(function () {
      var token = localStorage.getItem('era_token') || '';
      if (token && isTokenExpired(token)) redirectToLogin();
    }, 60000);
  }

  function initChrome(opts) {
    opts = opts || {};
    var root = opts.root || document;
    if (opts.moduleId) markActiveNav(opts.moduleId);
    mountNav(root);
    mountIcons(root);
    syncUserChip();
    if (global.EraChrome && global.EraChrome.ensureOfficeTopbarSwitcher) {
      global.EraChrome.ensureOfficeTopbarSwitcher();
    }
    wireSessionWatch();
    if (opts.menubar !== false && global.EraOfficeMenubar && global.EraOfficeMenubar.init) {
      var mb = opts.menubarRoot || root.querySelector('#menubar') || '#menubar';
      if (typeof mb === 'string' ? root.querySelector(mb) || document.querySelector(mb) : mb) {
        global.EraOfficeMenubar.init(mb, opts.menuHandlers || {});
      }
    }
    if (opts.commentsToggle != null) {
      wireCommentsToggle(!!opts.commentsToggle);
    }
  }

  /** Shared modal host — replaces window.prompt / confirm across Office apps. */
  function ensureUiDialog() {
    var dlg = document.getElementById('eraShellDialog');
    if (dlg) return dlg;
    dlg = document.createElement('dialog');
    dlg.id = 'eraShellDialog';
    dlg.className = 'era-shell-dialog';
    dlg.innerHTML =
      '<form method="dialog" class="era-shell-dialog-form">' +
      '<h3 class="era-shell-dialog-title"></h3>' +
      '<p class="era-shell-dialog-msg era-hint" hidden></p>' +
      '<label class="era-shell-dialog-field" hidden><span class="era-shell-dialog-label"></span>' +
      '<input class="era-input era-shell-dialog-input" type="text" autocomplete="off"/>' +
      '<textarea class="era-input era-shell-dialog-textarea" rows="4" hidden></textarea>' +
      '</label>' +
      '<div class="era-shell-dialog-choices" hidden role="listbox"></div>' +
      '<div class="era-shell-dialog-actions">' +
      '<button type="submit" value="cancel" class="era-btn era-shell-dialog-cancel">Cancel</button>' +
      '<button type="button" class="era-btn era-shell-dialog-copy" hidden>Copy</button>' +
      '<button type="submit" value="ok" class="era-btn era-btn-primary era-shell-dialog-ok">OK</button>' +
      '</div></form>';
    document.body.appendChild(dlg);
    return dlg;
  }

  function closeShellDialog(dlg, value) {
    if (!dlg || !dlg.open) return;
    try {
      dlg.close(value == null ? '' : String(value));
    } catch (_) {
      dlg.removeAttribute('open');
    }
  }

  /**
   * @param {{title?:string,message?:string,label?:string,value?:string,placeholder?:string,
   *   okLabel?:string,cancelLabel?:string,multiline?:boolean,danger?:boolean}} opts
   * @returns {Promise<string|null>}
   */
  function promptText(opts) {
    opts = opts || {};
    return new Promise(function (resolve) {
      var dlg = ensureUiDialog();
      var form = dlg.querySelector('form');
      var title = dlg.querySelector('.era-shell-dialog-title');
      var msg = dlg.querySelector('.era-shell-dialog-msg');
      var field = dlg.querySelector('.era-shell-dialog-field');
      var label = dlg.querySelector('.era-shell-dialog-label');
      var input = dlg.querySelector('.era-shell-dialog-input');
      var ta = dlg.querySelector('.era-shell-dialog-textarea');
      var choices = dlg.querySelector('.era-shell-dialog-choices');
      var okBtn = dlg.querySelector('.era-shell-dialog-ok');
      var cancelBtn = dlg.querySelector('.era-shell-dialog-cancel');
      var copyBtn = dlg.querySelector('.era-shell-dialog-copy');
      title.textContent = opts.title || 'Input';
      if (opts.message) {
        msg.hidden = false;
        msg.textContent = opts.message;
      } else {
        msg.hidden = true;
        msg.textContent = '';
      }
      field.hidden = false;
      choices.hidden = true;
      choices.innerHTML = '';
      copyBtn.hidden = true;
      label.textContent = opts.label || '';
      label.hidden = !opts.label;
      var multiline = !!opts.multiline;
      input.hidden = multiline;
      ta.hidden = !multiline;
      var active = multiline ? ta : input;
      active.value = opts.value != null ? String(opts.value) : '';
      active.placeholder = opts.placeholder || '';
      okBtn.textContent = opts.okLabel || 'OK';
      cancelBtn.textContent = opts.cancelLabel || 'Cancel';
      okBtn.classList.toggle('era-btn-danger', !!opts.danger);
      function finish(val) {
        form.onsubmit = null;
        dlg.oncancel = null;
        resolve(val);
      }
      form.onsubmit = function (ev) {
        ev.preventDefault();
        var submitter = ev.submitter;
        var v = submitter && submitter.value;
        if (v === 'cancel') {
          closeShellDialog(dlg, 'cancel');
          finish(null);
          return;
        }
        closeShellDialog(dlg, 'ok');
        finish(active.value);
      };
      dlg.oncancel = function (ev) {
        ev.preventDefault();
        closeShellDialog(dlg, 'cancel');
        finish(null);
      };
      if (typeof dlg.showModal === 'function') dlg.showModal();
      else dlg.setAttribute('open', '');
      setTimeout(function () {
        try {
          active.focus();
          if (active.select) active.select();
        } catch (_) {}
      }, 0);
    });
  }

  /**
   * @param {{title?:string,message?:string,okLabel?:string,cancelLabel?:string,danger?:boolean}} opts
   * @returns {Promise<boolean>}
   */
  function confirmAction(opts) {
    opts = opts || {};
    return new Promise(function (resolve) {
      var dlg = ensureUiDialog();
      var form = dlg.querySelector('form');
      var title = dlg.querySelector('.era-shell-dialog-title');
      var msg = dlg.querySelector('.era-shell-dialog-msg');
      var field = dlg.querySelector('.era-shell-dialog-field');
      var choices = dlg.querySelector('.era-shell-dialog-choices');
      var okBtn = dlg.querySelector('.era-shell-dialog-ok');
      var cancelBtn = dlg.querySelector('.era-shell-dialog-cancel');
      var copyBtn = dlg.querySelector('.era-shell-dialog-copy');
      title.textContent = opts.title || 'Confirm';
      msg.hidden = false;
      msg.textContent = opts.message || 'Are you sure?';
      field.hidden = true;
      choices.hidden = true;
      choices.innerHTML = '';
      copyBtn.hidden = true;
      okBtn.textContent = opts.okLabel || 'OK';
      cancelBtn.textContent = opts.cancelLabel || 'Cancel';
      okBtn.classList.toggle('era-btn-danger', !!opts.danger);
      function finish(ok) {
        form.onsubmit = null;
        dlg.oncancel = null;
        resolve(!!ok);
      }
      form.onsubmit = function (ev) {
        ev.preventDefault();
        var v = ev.submitter && ev.submitter.value;
        closeShellDialog(dlg, v || 'cancel');
        finish(v === 'ok');
      };
      dlg.oncancel = function (ev) {
        ev.preventDefault();
        closeShellDialog(dlg, 'cancel');
        finish(false);
      };
      if (typeof dlg.showModal === 'function') dlg.showModal();
      else dlg.setAttribute('open', '');
      setTimeout(function () {
        try {
          okBtn.focus();
        } catch (_) {}
      }, 0);
    });
  }

  /**
   * @param {{title?:string,message?:string,options:Array<{value:string,label:string,hint?:string}>,value?:string}} opts
   * @returns {Promise<string|null>}
   */
  function chooseOption(opts) {
    opts = opts || {};
    var options = opts.options || [];
    return new Promise(function (resolve) {
      var dlg = ensureUiDialog();
      var form = dlg.querySelector('form');
      var title = dlg.querySelector('.era-shell-dialog-title');
      var msg = dlg.querySelector('.era-shell-dialog-msg');
      var field = dlg.querySelector('.era-shell-dialog-field');
      var choices = dlg.querySelector('.era-shell-dialog-choices');
      var okBtn = dlg.querySelector('.era-shell-dialog-ok');
      var cancelBtn = dlg.querySelector('.era-shell-dialog-cancel');
      var copyBtn = dlg.querySelector('.era-shell-dialog-copy');
      title.textContent = opts.title || 'Choose';
      if (opts.message) {
        msg.hidden = false;
        msg.textContent = opts.message;
      } else {
        msg.hidden = true;
        msg.textContent = '';
      }
      field.hidden = true;
      copyBtn.hidden = true;
      choices.hidden = false;
      choices.innerHTML = '';
      var selected = opts.value != null ? String(opts.value) : options[0] ? options[0].value : '';
      options.forEach(function (opt) {
        var btn = document.createElement('button');
        btn.type = 'button';
        btn.className = 'era-shell-choice' + (opt.value === selected ? ' selected' : '');
        btn.setAttribute('data-value', opt.value);
        btn.innerHTML =
          '<strong>' +
          String(opt.label || opt.value)
            .replace(/&/g, '&amp;')
            .replace(/</g, '&lt;') +
          '</strong>' +
          (opt.hint
            ? '<span class="era-hint">' +
              String(opt.hint).replace(/&/g, '&amp;').replace(/</g, '&lt;') +
              '</span>'
            : '');
        btn.onclick = function () {
          selected = opt.value;
          choices.querySelectorAll('.era-shell-choice').forEach(function (el) {
            el.classList.toggle('selected', el.getAttribute('data-value') === selected);
          });
        };
        btn.ondblclick = function () {
          selected = opt.value;
          closeShellDialog(dlg, 'ok');
          finish(selected);
        };
        choices.appendChild(btn);
      });
      okBtn.textContent = opts.okLabel || 'OK';
      cancelBtn.textContent = opts.cancelLabel || 'Cancel';
      okBtn.classList.remove('era-btn-danger');
      function finish(val) {
        form.onsubmit = null;
        dlg.oncancel = null;
        resolve(val);
      }
      form.onsubmit = function (ev) {
        ev.preventDefault();
        var v = ev.submitter && ev.submitter.value;
        if (v === 'cancel') {
          closeShellDialog(dlg, 'cancel');
          finish(null);
          return;
        }
        closeShellDialog(dlg, 'ok');
        finish(selected || null);
      };
      dlg.oncancel = function (ev) {
        ev.preventDefault();
        closeShellDialog(dlg, 'cancel');
        finish(null);
      };
      if (typeof dlg.showModal === 'function') dlg.showModal();
      else dlg.setAttribute('open', '');
    });
  }

  /**
   * Share-link style: readonly field + Copy.
   * @returns {Promise<'copied'|'closed'|null>}
   */
  function promptCopy(opts) {
    opts = opts || {};
    return new Promise(function (resolve) {
      var dlg = ensureUiDialog();
      var form = dlg.querySelector('form');
      var title = dlg.querySelector('.era-shell-dialog-title');
      var msg = dlg.querySelector('.era-shell-dialog-msg');
      var field = dlg.querySelector('.era-shell-dialog-field');
      var label = dlg.querySelector('.era-shell-dialog-label');
      var input = dlg.querySelector('.era-shell-dialog-input');
      var ta = dlg.querySelector('.era-shell-dialog-textarea');
      var choices = dlg.querySelector('.era-shell-dialog-choices');
      var okBtn = dlg.querySelector('.era-shell-dialog-ok');
      var cancelBtn = dlg.querySelector('.era-shell-dialog-cancel');
      var copyBtn = dlg.querySelector('.era-shell-dialog-copy');
      title.textContent = opts.title || 'Copy link';
      if (opts.message) {
        msg.hidden = false;
        msg.textContent = opts.message;
      } else {
        msg.hidden = true;
      }
      field.hidden = false;
      choices.hidden = true;
      label.textContent = opts.label || 'Link';
      label.hidden = false;
      input.hidden = false;
      ta.hidden = true;
      input.value = opts.value != null ? String(opts.value) : '';
      input.readOnly = true;
      copyBtn.hidden = false;
      copyBtn.textContent = 'Copy';
      okBtn.textContent = opts.okLabel || 'Close';
      cancelBtn.hidden = true;
      function finish(val) {
        form.onsubmit = null;
        dlg.oncancel = null;
        copyBtn.onclick = null;
        cancelBtn.hidden = false;
        input.readOnly = false;
        resolve(val);
      }
      copyBtn.onclick = function () {
        var text = input.value || '';
        if (navigator.clipboard && navigator.clipboard.writeText) {
          navigator.clipboard.writeText(text).then(
            function () {
              copyBtn.textContent = 'Copied';
              toastStatus(document.getElementById('authStatus'), 'Link copied', false);
            },
            function () {
              try {
                input.select();
                document.execCommand('copy');
                copyBtn.textContent = 'Copied';
              } catch (_) {}
            }
          );
        } else {
          try {
            input.select();
            document.execCommand('copy');
            copyBtn.textContent = 'Copied';
          } catch (_) {}
        }
      };
      form.onsubmit = function (ev) {
        ev.preventDefault();
        closeShellDialog(dlg, 'ok');
        finish(copyBtn.textContent === 'Copied' ? 'copied' : 'closed');
      };
      dlg.oncancel = function (ev) {
        ev.preventDefault();
        closeShellDialog(dlg, 'cancel');
        finish(null);
      };
      if (typeof dlg.showModal === 'function') dlg.showModal();
      else dlg.setAttribute('open', '');
      setTimeout(function () {
        try {
          input.focus();
          input.select();
        } catch (_) {}
      }, 0);
    });
  }

  /**
   * Dismissible TE disclaimer (GAP-TE-G03). Persists dismissal in localStorage.
   * Banner: element with class era-banner-warn; optional [data-dismiss] button.
   */
  function wireTeDisclaimer(el, storageKey) {
    if (!el || !storageKey) return;
    try {
      if (localStorage.getItem(storageKey) === '1') {
        el.hidden = true;
        el.style.display = 'none';
        return;
      }
    } catch (_) {}
    el.hidden = false;
    el.style.display = '';
    var btn = el.querySelector('[data-dismiss]') || el.querySelector('.era-banner-dismiss');
    if (!btn) {
      btn = document.createElement('button');
      btn.type = 'button';
      btn.className = 'era-banner-dismiss';
      btn.setAttribute('data-dismiss', '1');
      btn.setAttribute('aria-label', 'Dismiss');
      btn.textContent = '✕';
      el.appendChild(btn);
    }
    btn.onclick = function () {
      try {
        localStorage.setItem(storageKey, '1');
      } catch (_) {}
      el.hidden = true;
      el.style.display = 'none';
    };
  }

  global.EraOfficeShell = {
    markActiveNav: markActiveNav,
    setStatus: setStatus,
    toastStatus: toastStatus,
    setSavePill: setSavePill,
    wireDocTitle: wireDocTitle,
    mountIcons: mountIcons,
    mountNav: mountNav,
    syncUserChip: syncUserChip,
    loginUrl: loginUrl,
    requireAuthOrRedirect: requireAuthOrRedirect,
    redirectToLogin: redirectToLogin,
    handleUnauthorized: handleUnauthorized,
    authFetch: authFetch,
    isTokenExpired: isTokenExpired,
    setCommentsOpen: setCommentsOpen,
    wireCommentsToggle: wireCommentsToggle,
    wireSessionWatch: wireSessionWatch,
    signOut: signOut,
    initChrome: initChrome,
    promptText: promptText,
    confirmAction: confirmAction,
    chooseOption: chooseOption,
    promptCopy: promptCopy,
    wireTeDisclaimer: wireTeDisclaimer,
  };

  if (global.EraOfficeMenubar) {
    global.EraOfficeShell.initMenubar = global.EraOfficeMenubar.init;
  }
})(typeof window !== 'undefined' ? window : globalThis);
