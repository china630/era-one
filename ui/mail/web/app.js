(function () {

  const params = new URLSearchParams(location.search);

  const tokenKey = 'era_mail_token';

  const emailKey = 'era_mail_email';

  const tenantKey = 'era_mail_tenant';



  if (params.get('code')) {

    exchangeCode(params.get('code'));

    return;

  }



  const token = localStorage.getItem(tokenKey);

  if (!token) {

    startLogin();

    return;

  }



  const claims = parseJwt(token);

  const email = localStorage.getItem(emailKey) || claims.email || 'alice@mail.gov.az';

  const tenant = localStorage.getItem(tenantKey) || claims.tenant_id || 't-demo';

  document.getElementById('user').textContent = email;
  if (window.EraChrome && EraChrome.mountAccount) {
    EraChrome.mountAccount(document.getElementById('user'), {
      label: email,
      showLamp: false,
      title: email,
    });
  }
  if (window.EraChrome && EraChrome.ensureOfficeTopbarSwitcher) {
    EraChrome.ensureOfficeTopbarSwitcher();
  }

  loadPolicy(tenant);

  loadMailbox(email, token);

  loadInbox(email, token);

  document.getElementById('composeBtn').onclick = () => {

    document.getElementById('compose').style.display = 'block';

  };

  document.getElementById('sendBtn').onclick = () => sendMail(email, token);



  async function startLogin() {

    const verifier = randomString(32);

    const digest = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(verifier));

    const challenge = b64url(new Uint8Array(digest));

    sessionStorage.setItem('pkce_verifier', verifier);

    const idBase = window.ERA_IDENTITY_URL || 'http://127.0.0.1:8160';

    const auth = new URL(idBase + '/oauth2/authorize');

    auth.searchParams.set('client_id', 'era-webmail');

    auth.searchParams.set('redirect_uri', location.origin + '/mail/callback');

    auth.searchParams.set('response_type', 'code');

    auth.searchParams.set('code_challenge', challenge);

    auth.searchParams.set('code_challenge_method', 'S256');

    auth.searchParams.set('state', 'mail');

    location.href = auth.toString();

  }



  async function exchangeCode(code) {

    const verifier = sessionStorage.getItem('pkce_verifier');

    const idBase = window.ERA_IDENTITY_URL || 'http://127.0.0.1:8160';

    const body = new URLSearchParams({

      grant_type: 'authorization_code',

      code: code,

      code_verifier: verifier,

      client_id: 'era-webmail',

      redirect_uri: location.origin + '/mail/callback',

    });

    const resp = await fetch(idBase + '/oauth2/token', { method: 'POST', body });

    const data = await resp.json();

    localStorage.setItem(tokenKey, data.access_token);

    const claims = parseJwt(data.access_token);

    if (claims.email) localStorage.setItem(emailKey, claims.email);

    if (claims.tenant_id) localStorage.setItem(tenantKey, claims.tenant_id);

    location.href = '/mail';

  }



  async function loadPolicy(tenant) {

    const resp = await fetch('/mail/api/policy?tenant_id=' + tenant);

    if (!resp.ok) return;

    const p = await resp.json();

    const el = document.getElementById('policy');

    el.hidden = false;

    el.textContent =

      'Quota policy: ' + p.quota_mb_per_user + ' MB per user · max attachment ' + p.max_attachment_size_mb + ' MB';

    window.__eraPolicy = p;

    updateSendButton();

  }



  async function loadMailbox(email, token) {

    const resp = await fetch('/mail/api/mailbox?email=' + encodeURIComponent(email), {

      headers: { Authorization: 'Bearer ' + token },

    });

    if (!resp.ok) return;

    window.__eraMailbox = await resp.json();

    updateSendButton();

  }



  function updateSendButton() {

    const btn = document.getElementById('sendBtn');

    const mb = window.__eraMailbox;

    if (!mb) return;

    const atQuota = mb.used_bytes >= mb.quota_bytes;

    btn.disabled = atQuota;

    btn.title = atQuota ? 'Mailbox quota full — sending disabled' : '';

  }



  async function loadInbox(email, token) {

    const resp = await fetch('/mail/api/messages?email=' + encodeURIComponent(email), {

      headers: { Authorization: 'Bearer ' + token },

    });

    const data = await resp.json();

    const ul = document.getElementById('inbox');

    ul.innerHTML = '';

    (data.messages || []).forEach((m) => {

      const li = document.createElement('li');

      const uid = m.uid != null ? m.uid : m.UID;

      li.textContent = (m.subject || m.Subject || '(no subject)') + ' — uid ' + uid;

      li.onclick = () => showMessage(uid, token);

      ul.appendChild(li);

    });

  }



  async function showMessage(uid, token) {

    const resp = await fetch('/mail/api/message?uid=' + encodeURIComponent(uid), {

      headers: { Authorization: 'Bearer ' + token },

    });

    if (!resp.ok) return;

    const msg = await resp.json();

    document.getElementById('messageDetail').hidden = false;

    document.getElementById('msgSubject').textContent = msg.subject || '(no subject)';

    document.getElementById('msgBody').textContent = msg.body || '(empty)';

    let attachBar = document.getElementById('msgAttachActions');
    if (!attachBar) {
      attachBar = document.createElement('div');
      attachBar.id = 'msgAttachActions';
      document.getElementById('messageDetail').appendChild(attachBar);
    }
    attachBar.innerHTML = '';
    const atts = msg.attachments || msg.Attachments || [];
    atts.forEach((a) => {
      const name = a.filename || a.Filename || a.name || '';
      const ct = a.content_type || a.ContentType || '';
      const oid = a.drive_object_id || a.DriveObjectID || a.object_id || '';
      if (!oid || !(name.toLowerCase().endsWith('.erad') || name.toLowerCase().endsWith('.docx'))) {
        return;
      }
      const btn = document.createElement('button');
      btn.type = 'button';
      btn.textContent = 'Редактировать в Documents';
      btn.onclick = async () => {
        const r = await fetch('/mail/api/documents/edit-link', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json', Authorization: 'Bearer ' + token },
          body: JSON.stringify({ drive_object_id: oid, filename: name, content_type: ct }),
        });
        if (!r.ok) return;
        const out = await r.json();
        if (out.url) location.href = out.url;
      };
      attachBar.appendChild(btn);
    });

  }



  async function sendMail(from, token) {

    const to = document.getElementById('to').value;

    const subject = document.getElementById('subject').value;

    const body = document.getElementById('body').value;

    await fetch('/mail/api/send', {

      method: 'POST',

      headers: { 'Content-Type': 'application/json', Authorization: 'Bearer ' + token },

      body: JSON.stringify({ from, to, subject, body }),

    });

    loadInbox(from, token);

    loadMailbox(from, token);

  }



  function parseJwt(token) {

    try {

      const payload = token.split('.')[1];

      const json = atob(payload.replace(/-/g, '+').replace(/_/g, '/'));

      return JSON.parse(json);

    } catch (_) {

      return {};

    }

  }



  function randomString(n) {

    const a = new Uint8Array(n);

    crypto.getRandomValues(a);

    return b64url(a);

  }



  function b64url(bytes) {

    let str = '';

    bytes.forEach((b) => (str += String.fromCharCode(b)));

    return btoa(str).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');

  }

})();

