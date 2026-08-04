/* Shared ERA chrome helpers (Theme Matrix Phase B).
 * Sync to control-shell via scripts/sync-era-chrome.ps1 */
(function (global) {
  var PRODUCTS = [
    { id: 'control', label: 'ERA Control', href: '/ui/control/' },
    { id: 'comms', label: 'ERA Communications', href: '/mail/' },
    { id: 'office', label: 'ERA Office', href: '/drive/' },
  ];

  var LINE_LABELS = {
    control: 'ERA Control',
    comms: 'ERA Communications',
    office: 'ERA Office',
  };

  function currentLine() {
    var line = (document.body && document.body.getAttribute('data-line')) || '';
    if (line) return line;
    var path = location.pathname || '';
    if (path.indexOf('/ui/control') === 0) return 'control';
    if (path.indexOf('/mail') === 0 || path.indexOf('/chat') === 0) return 'comms';
    return 'office';
  }

  /** Lab default: all enabled. Optional opts.licensed = {control:true,...} */
  function licensedMap(opts) {
    opts = opts || {};
    if (opts.licensed && typeof opts.licensed === 'object') return opts.licensed;
    // TODO license gate: wire /api/v1/shell/config or products API when available
    return { control: true, comms: true, office: true };
  }

  function mountBrand(el, opts) {
    if (!el) return null;
    opts = opts || {};
    var line = opts.line || currentLine();
    var title = opts.title || LINE_LABELS[line] || 'ERA One';
    var subtitle = opts.subtitle || '';
    var href = opts.href || '#';
    el.classList.add('era-brand');
    if (el.tagName === 'A' || el.tagName === 'a') {
      el.setAttribute('href', href);
    }
    el.innerHTML = '';
    var mark = document.createElement('span');
    mark.className = 'era-brand-mark';
    mark.textContent = title;
    el.appendChild(mark);
    if (subtitle) {
      var sub = document.createElement('span');
      sub.className = 'era-brand-mod';
      if (el.tagName !== 'A' && el.tagName !== 'a') {
        // Control sidebar uses nested <span> for subtitle
        sub = document.createElement('span');
      }
      sub.textContent = subtitle;
      el.appendChild(sub);
    }
    return el;
  }

  function mountAccount(el, opts) {
    if (!el) return null;
    opts = opts || {};
    el.classList.add('era-user-chip');
    var label = opts.label || opts.role || '—';
    var lamp = el.querySelector('.era-user-lamp');
    var text = el.querySelector('.era-user-label');
    if (!text) {
      el.textContent = '';
      if (opts.showLamp !== false && (opts.online != null || opts.showLamp)) {
        lamp = document.createElement('span');
        lamp.className = 'era-user-lamp' + (opts.online ? ' on' : ' off');
        el.appendChild(lamp);
      }
      text = document.createElement('span');
      text.className = 'era-user-label';
      el.appendChild(text);
    }
    text.textContent = label;
    if (opts.title) el.title = opts.title;
    else el.title = label;
    return el;
  }

  function closeMenus(except) {
    document.querySelectorAll('.era-product-switcher-menu').forEach(function (m) {
      if (m !== except) m.hidden = true;
    });
  }

  function mountSwitcher(container, opts) {
    if (!container) return null;
    opts = opts || {};
    var existing = container.querySelector('.era-product-switcher');
    if (existing && !opts.replace) return existing;

    var wrap = document.createElement('div');
    wrap.className = 'era-product-switcher';
    wrap.setAttribute('data-era-chrome', 'switcher');

    var btn = document.createElement('button');
    btn.type = 'button';
    btn.setAttribute('aria-haspopup', 'menu');
    btn.setAttribute('aria-expanded', 'false');
    btn.textContent = 'ERA One';

    var menu = document.createElement('ul');
    menu.className = 'era-product-switcher-menu';
    menu.setAttribute('role', 'menu');
    menu.hidden = true;

    var line = currentLine();
    var lic = licensedMap(opts);

    PRODUCTS.forEach(function (p) {
      var li = document.createElement('li');
      li.setAttribute('role', 'none');
      var enabled = lic[p.id] !== false;
      if (!enabled) {
        var span = document.createElement('span');
        span.className = 'disabled';
        span.textContent = p.label;
        span.title = 'Not licensed';
        span.setAttribute('role', 'menuitem');
        span.setAttribute('aria-disabled', 'true');
        li.appendChild(span);
      } else {
        var a = document.createElement('a');
        a.href = p.href;
        a.textContent = p.label;
        a.setAttribute('role', 'menuitem');
        if (p.id === line) a.setAttribute('aria-current', 'page');
        li.appendChild(a);
      }
      menu.appendChild(li);
    });

    btn.addEventListener('click', function (e) {
      e.stopPropagation();
      var open = menu.hidden;
      closeMenus(menu);
      menu.hidden = !open;
      btn.setAttribute('aria-expanded', open ? 'true' : 'false');
    });

    wrap.appendChild(btn);
    wrap.appendChild(menu);

    if (existing) existing.replaceWith(wrap);
    else if (opts.prepend) container.insertBefore(wrap, container.firstChild);
    else container.appendChild(wrap);

    if (!global.__eraChromeSwitcherDocClick) {
      global.__eraChromeSwitcherDocClick = true;
      document.addEventListener('click', function () {
        closeMenus(null);
        document.querySelectorAll('.era-product-switcher > button').forEach(function (b) {
          b.setAttribute('aria-expanded', 'false');
        });
      });
    }

    return wrap;
  }

  /** Inject switcher into Office topbar-right (before account chip). */
  function ensureOfficeTopbarSwitcher() {
    var right = document.querySelector('.era-topbar-right');
    if (!right) return null;
    if (right.querySelector('.era-product-switcher')) return right.querySelector('.era-product-switcher');
    var chip = right.querySelector('.era-user-chip, #userChip');
    var wrap = mountSwitcher(right, { prepend: !chip });
    if (chip && wrap && wrap.parentNode === right && chip.previousSibling !== wrap) {
      right.insertBefore(wrap, chip);
    }
    return wrap;
  }

  function boot() {
    document.documentElement.classList.add('era-chrome-focus');
    if (document.body && document.body.classList.contains('era-app')) {
      ensureOfficeTopbarSwitcher();
    }
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', boot);
  } else {
    boot();
  }

  global.EraChrome = {
    mountBrand: mountBrand,
    mountAccount: mountAccount,
    mountSwitcher: mountSwitcher,
    ensureOfficeTopbarSwitcher: ensureOfficeTopbarSwitcher,
    products: PRODUCTS,
    currentLine: currentLine,
  };
})(typeof window !== 'undefined' ? window : globalThis);
