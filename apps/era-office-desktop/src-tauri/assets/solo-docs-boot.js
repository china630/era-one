/* Solo Office — auth bypass, skin, File open/save → local bridge (no extra toolbar). */
(function () {
  var exp = Math.floor(Date.now() / 1000) + 86400 * 3650;
  var payload = btoa(
    JSON.stringify({
      sub: 'solo-user',
      tenant_id: 'solo',
      name: 'Solo',
      exp: exp,
    })
  )
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
    .replace(/=+$/, '');
  var header = btoa(JSON.stringify({ alg: 'none', typ: 'JWT' }))
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
    .replace(/=+$/, '');
  try {
    localStorage.setItem('era_token', header + '.' + payload + '.');
  } catch (_) {}

  function patchShell() {
    if (!window.EraOfficeShell) return false;
    EraOfficeShell.requireAuthOrRedirect = function () {
      return true;
    };
    EraOfficeShell.redirectToLogin = function () {};
    EraOfficeShell.handleUnauthorized = function () {
      return false;
    };
    return true;
  }

  function markSolo() {
    if (document.body) document.body.classList.add('era-solo');
  }

  function productPrefix() {
    var p = location.pathname;
    if (p.indexOf('/tables') === 0) return 'tables';
    if (p.indexOf('/presentations') === 0) return 'presentations';
    if (p.indexOf('/projects') === 0) return 'projects';
    if (p.indexOf('/docs') === 0) return 'docs';
    return 'hub';
  }

  function call(path, thenReload) {
    return fetch(path, { method: 'POST' })
      .then(function (r) {
        if (!r.ok) throw new Error(path + ' ' + r.status);
        return r.json().catch(function () {
          return {};
        });
      })
      .then(function (data) {
        if (thenReload) {
          var id = data.drive_object_id || 'solo';
          var pref = productPrefix();
          if (pref === 'docs') location.href = '/docs/' + id;
          else if (pref === 'tables') location.href = '/tables/' + id;
          else if (pref === 'presentations') location.href = '/presentations/' + id;
          else if (pref === 'projects') location.href = '/projects/' + id;
          else location.reload();
        }
      })
      .catch(function (e) {
        console.warn(e);
      });
  }

  function productFileQs() {
    var pref = productPrefix();
    if (pref === 'hub') return 'docs';
    return pref;
  }

  function patchProductMenu(handlersName) {
    var tries = 0;
    (function tick() {
      tries++;
      var handlers = window[handlersName];
      if (typeof handlers === 'object' && handlers) {
        var qs = '?product=' + encodeURIComponent(productFileQs());
        handlers['file.open'] = function () {
          call('/api/v1/solo/file/open' + qs, true);
        };
        handlers['file.save'] = function () {
          call('/api/v1/solo/file/save' + qs, false);
        };
        var fileMenu = document.querySelector('.era-menu-panel');
        if (fileMenu && !document.querySelector('[data-cmd="file.saveAs"]')) {
          var saveItem = document.querySelector('[data-cmd="file.save"]');
          if (saveItem) {
            var btn = document.createElement('button');
            btn.type = 'button';
            btn.className = 'era-menu-item';
            btn.setAttribute('data-cmd', 'file.saveAs');
            btn.innerHTML = '<span>Save As…</span>';
            saveItem.parentNode.insertBefore(btn, saveItem.nextSibling);
          }
        }
        handlers['file.saveAs'] = function () {
          call('/api/v1/solo/file/save-as' + qs, false);
        };
        // Map openDrive to local open in Solo
        handlers['file.openDrive'] = handlers['file.open'];
        return;
      }
      if (tries < 40) setTimeout(tick, 50);
    })();
  }

  function patchDocsSnapshot() {
    var tries = 0;
    (function tick() {
      tries++;
      if (typeof snapshotDoc === 'function') {
        window.snapshotDoc = async function () {
          if (typeof docId === 'undefined' || !docId) return;
          var idn =
            typeof identity === 'function'
              ? identity()
              : { tenantId: 'solo', userId: 'solo-user' };
          var body = { tenant_id: idn.tenantId, user_id: idn.userId };
          if (typeof docState !== 'undefined') body.document = docState;
          var res = await fetch('/api/v1/docs/' + docId + '/snapshot', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(body),
          });
          return res.ok;
        };
        return;
      }
      if (tries < 40) setTimeout(tick, 50);
    })();
  }

  function patchDocsMenu() {
    var tries = 0;
    (function tick() {
      tries++;
      if (typeof docsMenuHandlers === 'object' && docsMenuHandlers) {
        var qs = '?product=docs';
        docsMenuHandlers['file.open'] = function () {
          call('/api/v1/solo/file/open' + qs, true);
        };
        docsMenuHandlers['file.save'] = function () {
          Promise.resolve(window.snapshotDoc && window.snapshotDoc())
            .then(function () {
              return call('/api/v1/solo/file/save' + qs, false);
            })
            .catch(function () {});
        };
        var fileMenu = document.querySelector('.era-menu-panel');
        if (fileMenu && !document.querySelector('[data-cmd="file.saveAs"]')) {
          var saveItem = document.querySelector('[data-cmd="file.save"]');
          if (saveItem) {
            var btn = document.createElement('button');
            btn.type = 'button';
            btn.className = 'era-menu-item';
            btn.setAttribute('data-cmd', 'file.saveAs');
            btn.innerHTML = '<span>Save As…</span>';
            saveItem.parentNode.insertBefore(btn, saveItem.nextSibling);
          }
        }
        docsMenuHandlers['file.saveAs'] = function () {
          Promise.resolve(window.snapshotDoc && window.snapshotDoc())
            .then(function () {
              return call('/api/v1/solo/file/save-as' + qs, false);
            })
            .catch(function () {});
        };
        return;
      }
      if (tries < 40) setTimeout(tick, 50);
    })();
  }

  function wireNav() {
    document.addEventListener(
      'click',
      function (e) {
        var a = e.target && e.target.closest && e.target.closest('a.era-nav a, .era-nav a, a.era-brand');
        if (!a) return;
        var href = a.getAttribute('href') || '';
        // Keep local product routes; block mail/AI/drive deep tenant flows
        if (href === '/mail/' || href.indexOf('/office-ai') === 0) {
          e.preventDefault();
          return;
        }
        if (href === '/drive/') {
          e.preventDefault();
          location.href = '/';
        }
      },
      true
    );
  }

  function patchMenubarForSolo() {
    if (!window.EraOfficeMenubar || EraOfficeMenubar.__soloFilePatched) return !!window.EraOfficeMenubar;
    var orig = EraOfficeMenubar.init.bind(EraOfficeMenubar);
    EraOfficeMenubar.init = function (sel, handlers) {
      var qs = '?product=' + encodeURIComponent(productFileQs());
      var h = Object.assign({}, handlers || {});
      h['file.open'] = function () {
        call('/api/v1/solo/file/open' + qs, true);
      };
      h['file.openDrive'] = h['file.open'];
      var prevSave = h['file.save'];
      h['file.save'] = function () {
        if (productPrefix() === 'docs' && window.snapshotDoc) {
          Promise.resolve(window.snapshotDoc())
            .then(function () {
              return call('/api/v1/solo/file/save' + qs, false);
            })
            .catch(function () {});
        } else if (typeof prevSave === 'function') {
          Promise.resolve(prevSave())
            .then(function () {
              return call('/api/v1/solo/file/save' + qs, false);
            })
            .catch(function () {
              call('/api/v1/solo/file/save' + qs, false);
            });
        } else {
          call('/api/v1/solo/file/save' + qs, false);
        }
      };
      h['file.saveAs'] = function () {
        if (productPrefix() === 'docs' && window.snapshotDoc) {
          Promise.resolve(window.snapshotDoc())
            .then(function () {
              return call('/api/v1/solo/file/save-as' + qs, false);
            })
            .catch(function () {});
        } else if (productPrefix() === 'presentations' && typeof prevSave === 'function') {
          Promise.resolve(prevSave())
            .then(function () {
              return call('/api/v1/solo/file/save-as' + qs, false);
            })
            .catch(function () {
              call('/api/v1/solo/file/save-as' + qs, false);
            });
        } else {
          call('/api/v1/solo/file/save-as' + qs, false);
        }
      };
      return orig(sel, h);
    };
    EraOfficeMenubar.__soloFilePatched = true;
    return true;
  }

  markSolo();
  document.addEventListener('DOMContentLoaded', markSolo);
  wireNav();
  patchMenubarForSolo();

  var n = 0;
  (function waitShell() {
    patchMenubarForSolo();
    if (patchShell() || ++n > 100) {
      var pref = productPrefix();
      if (pref === 'docs') {
        patchDocsSnapshot();
        patchDocsMenu();
      }
      return;
    }
    setTimeout(waitShell, 20);
  })();
})();
