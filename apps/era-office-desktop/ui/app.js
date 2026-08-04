/* ERA Office desktop launcher — Solo hub / SKU product or Corporate Workspace. */
(function () {
  const panelBoot = document.getElementById('panelBoot');
  const corpSetup = document.getElementById('corpSetup');
  const serverUrlEl = document.getElementById('serverUrl');
  const bootStatus = document.getElementById('bootStatus');

  function invoke(cmd, args) {
    const core = window.__TAURI__ && window.__TAURI__.core;
    if (!core || !core.invoke) {
      return Promise.reject(new Error('Tauri API not available (open via era-office-desktop)'));
    }
    return core.invoke(cmd, args || {});
  }

  function setBootStatus(msg, isErr) {
    bootStatus.textContent = msg || '';
    bootStatus.className = 'status' + (isErr ? ' err' : '');
  }

  function showBoot(opts) {
    panelBoot.hidden = false;
    corpSetup.hidden = !(opts && opts.corpForm);
    document.title = 'ERA Office';
  }

  async function openSoloEntry() {
    setBootStatus('Opening Solo…', false);
    try {
      const href = await invoke('solo_entry_go');
      setBootStatus('Solo: ' + href, false);
    } catch (e) {
      setBootStatus(String(e), true);
    }
  }

  document.getElementById('pickSolo').onclick = async () => {
    try {
      await invoke('config_set', { profile: 'solo', serverUrl: null });
      await openSoloEntry();
    } catch (e) {
      setBootStatus(String(e), true);
    }
  };

  document.getElementById('pickCorp').onclick = () => {
    showBoot({ corpForm: true });
    setBootStatus('', false);
  };

  document.getElementById('corpBack').onclick = () => {
    corpSetup.hidden = true;
    setBootStatus('', false);
  };

  document.getElementById('corpConnect').onclick = async () => {
    const url = (serverUrlEl.value || '').trim();
    if (!url) {
      setBootStatus('Enter Workspace URL', true);
      return;
    }
    try {
      setBootStatus('Saving…', false);
      await invoke('config_set', { profile: 'corporate', serverUrl: url });
      const href = await invoke('corp_go', { path: null });
      setBootStatus('Opening ' + href + '…', false);
    } catch (e) {
      setBootStatus(String(e), true);
    }
  };

  invoke('config_get')
    .then((cfg) => {
      const ready =
        cfg.profile === 'corporate' && cfg.server_url && String(cfg.server_url).length > 0;
      if (ready) {
        showBoot({ corpForm: true });
        serverUrlEl.value = cfg.server_url || '';
        setBootStatus('Connecting to Workspace…', false);
        return invoke('corp_go', { path: null }).catch((e) => setBootStatus(String(e), true));
      }
      if (cfg.profile === 'corporate') {
        showBoot({ corpForm: true });
        serverUrlEl.value = cfg.server_url || '';
        return;
      }
      if (cfg.sku && cfg.sku !== 'suite') {
        return openSoloEntry();
      }
      if (cfg.first_run) {
        showBoot({ corpForm: false });
        return;
      }
      return openSoloEntry();
    })
    .catch((e) => {
      showBoot({ corpForm: false });
      setBootStatus(String(e), true);
    });
})();
