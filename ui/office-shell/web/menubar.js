/* Shared Google-like menubar with nested submenus.
 * Click once to open → subsequent top-level menus open on hover (sticky mode).
 */
(function (global) {
  function closeAll(root) {
    (root || document).querySelectorAll('.era-menubar .era-menu.open').forEach(function (m) {
      m.classList.remove('open');
      var btn = m.querySelector('.era-menu-btn');
      if (btn) btn.setAttribute('aria-expanded', 'false');
    });
    (root || document).querySelectorAll('.era-menubar .era-submenu.open').forEach(function (s) {
      s.classList.remove('open');
    });
  }

  function closeSubmenus(panel) {
    if (!panel) return;
    panel.querySelectorAll('.era-submenu.open').forEach(function (s) {
      s.classList.remove('open');
    });
  }

  function openMenu(bar, menu) {
    if (!menu) return;
    closeAll(bar);
    menu.classList.add('open');
    var btn = menu.querySelector('.era-menu-btn');
    if (btn) btn.setAttribute('aria-expanded', 'true');
  }

  function setActive(bar, on) {
    if (on) bar.classList.add('era-menubar-active');
    else bar.classList.remove('era-menubar-active');
  }

  function isActive(bar) {
    return bar.classList.contains('era-menubar-active');
  }

  function initMenubar(root, handlers) {
    var bar = typeof root === 'string' ? document.querySelector(root) : root;
    if (!bar) return;
    handlers = handlers || {};

    bar.querySelectorAll('.era-menu-btn').forEach(function (btn) {
      btn.addEventListener('click', function (ev) {
        ev.stopPropagation();
        var menu = btn.closest('.era-menu');
        var wasOpen = menu.classList.contains('open');
        if (wasOpen && isActive(bar)) {
          // Second click on same menu closes and leaves sticky mode.
          closeAll(bar);
          setActive(bar, false);
          return;
        }
        openMenu(bar, menu);
        setActive(bar, true);
      });

      btn.addEventListener('mouseenter', function () {
        if (!isActive(bar)) return;
        var menu = btn.closest('.era-menu');
        if (menu && !menu.classList.contains('open')) openMenu(bar, menu);
      });
    });

    // Hover into an already-open menu keeps sticky mode; hover sibling switches.
    bar.querySelectorAll('.era-menu').forEach(function (menu) {
      menu.addEventListener('mouseenter', function () {
        if (!isActive(bar)) return;
        if (!menu.classList.contains('open')) openMenu(bar, menu);
      });
    });

    bar.querySelectorAll('.era-submenu-btn').forEach(function (btn) {
      btn.addEventListener('click', function (ev) {
        ev.stopPropagation();
        ev.preventDefault();
        var sub = btn.closest('.era-submenu');
        if (!sub) return;
        var panel = sub.closest('.era-menu-panel');
        var wasOpen = sub.classList.contains('open');
        closeSubmenus(panel);
        if (!wasOpen) sub.classList.add('open');
      });
      btn.addEventListener('mouseenter', function () {
        if (!isActive(bar)) return;
        var sub = btn.closest('.era-submenu');
        if (!sub) return;
        var panel = sub.closest('.era-menu-panel');
        closeSubmenus(panel);
        sub.classList.add('open');
      });
    });

    bar.querySelectorAll('.era-menu-item[data-cmd]').forEach(function (item) {
      item.addEventListener('click', function (ev) {
        if (item.classList.contains('era-submenu-btn')) return;
        ev.stopPropagation();
        if (item.disabled) return;
        var cmd = item.getAttribute('data-cmd');
        closeAll(bar);
        setActive(bar, false);
        if (typeof handlers[cmd] === 'function') {
          handlers[cmd](item);
        } else if (typeof handlers.onCommand === 'function') {
          handlers.onCommand(cmd, item);
        }
      });
    });

    if (global.EraOfficeIcons && global.EraOfficeIcons.mountMenuIcons) {
      global.EraOfficeIcons.mountMenuIcons(bar);
    }

    document.addEventListener('click', function () {
      closeAll(bar);
      setActive(bar, false);
    });
    document.addEventListener('keydown', function (ev) {
      if (ev.key === 'Escape') {
        closeAll(bar);
        setActive(bar, false);
      }
    });
  }

  function menuItem(cmd, label, opts) {
    opts = opts || {};
    var disabled = opts.disabled ? ' disabled' : '';
    var title = opts.title ? ' title="' + opts.title.replace(/"/g, '&quot;') + '"' : '';
    var hint = opts.hint ? '<span class="hint">' + opts.hint + '</span>' : '';
    return (
      '<button type="button" class="era-menu-item" data-cmd="' +
      cmd +
      '"' +
      disabled +
      title +
      '>' +
      '<span>' +
      label +
      '</span>' +
      hint +
      '</button>'
    );
  }

  function sep() {
    return '<div class="era-menu-sep" role="separator"></div>';
  }

  function submenu(label, itemsHtml, opts) {
    opts = opts || {};
    var iconAttr = opts.icon ? ' data-icon="' + opts.icon + '"' : '';
    return (
      '<div class="era-submenu">' +
      '<button type="button" class="era-menu-item era-submenu-btn" aria-haspopup="true"' +
      iconAttr +
      '>' +
      '<span>' +
      label +
      '</span>' +
      '<span class="era-submenu-caret" aria-hidden="true">›</span>' +
      '</button>' +
      '<div class="era-submenu-panel" role="menu">' +
      itemsHtml +
      '</div></div>'
    );
  }

  function menu(label, itemsHtml) {
    return (
      '<div class="era-menu">' +
      '<button type="button" class="era-menu-btn" aria-expanded="false" aria-haspopup="true">' +
      label +
      '</button>' +
      '<div class="era-menu-panel" role="menu">' +
      itemsHtml +
      '</div></div>'
    );
  }

  global.EraOfficeMenubar = {
    init: initMenubar,
    closeAll: closeAll,
    menuItem: menuItem,
    sep: sep,
    menu: menu,
    submenu: submenu,
  };
})(typeof window !== 'undefined' ? window : globalThis);
