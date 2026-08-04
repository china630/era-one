/* Shared Google Docs–style toolbar chrome: tooltips + button menus. */
(function (global) {
  var tipEl = null;
  var tipTimer = null;

  function ensureTip() {
    if (tipEl) return tipEl;
    tipEl = document.createElement('div');
    tipEl.className = 'era-tip';
    tipEl.setAttribute('role', 'tooltip');
    tipEl.hidden = true;
    document.body.appendChild(tipEl);
    return tipEl;
  }

  function hideTip() {
    clearTimeout(tipTimer);
    if (tipEl) tipEl.hidden = true;
  }

  function showTip(target) {
    var label =
      target.getAttribute('data-tip') ||
      target.getAttribute('aria-label') ||
      target.getAttribute('title') ||
      '';
    if (!label) return;
    var shortcut = target.getAttribute('data-shortcut') || '';
    var el = ensureTip();
    el.innerHTML =
      '<span class="era-tip-label"></span>' +
      (shortcut ? '<span class="era-tip-shortcut"></span>' : '');
    el.querySelector('.era-tip-label').textContent = label;
    if (shortcut) el.querySelector('.era-tip-shortcut').textContent = shortcut;
    if (target.getAttribute('title') && !target.getAttribute('data-tip-keep-title')) {
      target.setAttribute('data-native-title', target.getAttribute('title'));
      target.removeAttribute('title');
    }
    var rect = target.getBoundingClientRect();
    el.hidden = false;
    var tw = el.offsetWidth;
    var th = el.offsetHeight;
    var left = rect.left + rect.width / 2 - tw / 2;
    left = Math.max(8, Math.min(left, window.innerWidth - tw - 8));
    var top = rect.bottom + 8;
    if (top + th > window.innerHeight - 8) top = rect.top - th - 8;
    el.style.left = left + 'px';
    el.style.top = top + 'px';
  }

  function wireTooltips(root) {
    var scope = root || document;
    scope.addEventListener(
      'mouseover',
      function (e) {
        var t = e.target.closest(
          '.era-icon-btn[title], .era-icon-btn[data-tip], .era-btn[title], .era-btn[data-tip], [data-tip]'
        );
        if (!t || t.disabled) return;
        clearTimeout(tipTimer);
        tipTimer = setTimeout(function () {
          showTip(t);
        }, 400);
      },
      true
    );
    scope.addEventListener(
      'mouseout',
      function (e) {
        var t = e.target.closest('.era-icon-btn, .era-btn, [data-tip]');
        if (!t) return;
        hideTip();
      },
      true
    );
    scope.addEventListener('mousedown', hideTip, true);
    scope.addEventListener('scroll', hideTip, true);
  }

  function closeAllMenus(except) {
    document.querySelectorAll('.era-btn-menu.open').forEach(function (m) {
      if (m !== except) {
        m.classList.remove('open');
        var btn = m.querySelector('.era-btn-menu-trigger');
        if (btn) btn.setAttribute('aria-expanded', 'false');
      }
    });
  }

  function wireButtonMenus(root) {
    var scope = root || document;
    scope.querySelectorAll('.era-btn-menu').forEach(function (menu) {
      if (menu._eraMenuWired) return;
      menu._eraMenuWired = true;
      var trigger = menu.querySelector('.era-btn-menu-trigger');
      var panel = menu.querySelector('.era-btn-menu-panel');
      if (!trigger || !panel) return;
      trigger.setAttribute('aria-haspopup', 'true');
      trigger.setAttribute('aria-expanded', 'false');
      trigger.addEventListener('click', function (e) {
        e.preventDefault();
        e.stopPropagation();
        var open = !menu.classList.contains('open');
        closeAllMenus(menu);
        menu.classList.toggle('open', open);
        trigger.setAttribute('aria-expanded', open ? 'true' : 'false');
      });
      panel.addEventListener('click', function (e) {
        var item = e.target.closest('[data-menu-value], [data-cmd], button');
        if (!item || item.disabled) return;
        if (!item.hasAttribute('data-keep-open')) {
          menu.classList.remove('open');
          trigger.setAttribute('aria-expanded', 'false');
        }
      });
    });
    if (!document._eraBtnMenuDocClose) {
      document._eraBtnMenuDocClose = true;
      document.addEventListener('click', function () {
        closeAllMenus(null);
      });
      document.addEventListener('keydown', function (e) {
        if (e.key === 'Escape') closeAllMenus(null);
      });
    }
  }

  /**
   * Keep editor selection when clicking toolbar buttons.
   * Do NOT preventDefault on native select / text inputs — that blocks dropdowns.
   */
  function preserveSelectionOnToolbar(root) {
    var scope = root || document;
    scope.querySelectorAll('.era-toolbar, .era-toolbar-rail').forEach(function (bar) {
      if (bar._eraPreserveSel) return;
      bar._eraPreserveSel = true;
      bar.addEventListener('mousedown', function (e) {
        var t = e.target.closest(
          'button, .era-icon-btn, .era-btn, input[type=color], .era-menu-preset'
        );
        if (!t) return;
        if (t.matches('select, option, input, textarea, label')) return;
        if (e.target.closest('select, input, textarea')) return;
        e.preventDefault();
      });
    });
  }

  function initToolbarChrome(root) {
    wireTooltips(root);
    wireButtonMenus(root);
    preserveSelectionOnToolbar(root);
  }

  global.EraOfficeToolbar = {
    init: initToolbarChrome,
    wireTooltips: wireTooltips,
    wireButtonMenus: wireButtonMenus,
    closeMenus: closeAllMenus,
  };
})(typeof window !== 'undefined' ? window : globalThis);
