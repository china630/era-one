(function () {
  var params = new URLSearchParams(location.search);
  var next = params.get('next') || '/drive/';
  if (!next.startsWith('/') || next.startsWith('//')) next = '/drive/';

  var PRODUCT_LABELS = {
    office: 'ERA Office',
    comms: 'ERA Communications',
    control: 'ERA Control',
  };

  function inferProduct() {
    var explicit = (params.get('product') || '').toLowerCase();
    if (PRODUCT_LABELS[explicit]) return explicit;
    var path = next.split('?')[0];
    if (path.indexOf('/mail') === 0 || path.indexOf('/chat') === 0) return 'comms';
    if (path.indexOf('/ui/control') === 0 || path.indexOf('/control') === 0) return 'control';
    return 'office';
  }

  var product = inferProduct();
  document.body.setAttribute('data-product', product);
  var productEl = document.querySelector('.era-login-product');
  if (productEl) productEl.textContent = PRODUCT_LABELS[product];
  document.title = 'Sign in — ' + PRODUCT_LABELS[product];

  var stepEmail = document.getElementById('stepEmail');
  var stepPassword = document.getElementById('stepPassword');
  var stepCreate = document.getElementById('stepCreate');
  var titleEl = document.getElementById('loginTitle');
  var subEl = document.getElementById('loginSubtitle');
  var emailInput = document.getElementById('email');
  var passwordInput = document.getElementById('password');
  var chip = document.getElementById('backEmail');

  var pendingEmail = '';

  function showError(id, msg) {
    var el = document.getElementById(id);
    if (!el) return;
    if (!msg) {
      el.hidden = true;
      el.textContent = '';
      return;
    }
    el.hidden = false;
    el.textContent = msg;
  }

  function showStep(name) {
    stepEmail.hidden = name !== 'email';
    stepPassword.hidden = name !== 'password';
    stepCreate.hidden = name !== 'create';
    if (name === 'email') {
      titleEl.textContent = 'Sign in';
      subEl.textContent = 'Use your ERA account';
      showError('emailError', '');
      setTimeout(function () { emailInput.focus(); }, 0);
    } else if (name === 'password') {
      titleEl.textContent = 'Welcome';
      subEl.textContent = 'Enter your password to continue';
      chip.textContent = pendingEmail;
      chip.setAttribute('data-initial', (pendingEmail[0] || '?').toUpperCase());
      showError('passwordError', '');
      passwordInput.value = '';
      setTimeout(function () { passwordInput.focus(); }, 0);
    } else {
      titleEl.textContent = 'Create your ERA Account';
      subEl.textContent = 'One account for Drive, Docs, Tables, and more';
      showError('createError', '');
      setTimeout(function () { document.getElementById('regEmail').focus(); }, 0);
    }
  }

  function safeNext() {
    try {
      var u = new URL(next, location.origin);
      if (u.origin !== location.origin) return '/drive/';
      return u.pathname + u.search + u.hash;
    } catch (_) {
      return '/drive/';
    }
  }

  async function signIn(email, password) {
    var res = await fetch('/oauth2/staging/token', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: email, password: password }),
    });
    if (!res.ok) {
      var t = await res.text();
      throw new Error(res.status === 401 ? 'Wrong email or password' : (t || 'Sign-in failed (' + res.status + ')'));
    }
    var data = await res.json();
    if (!data.access_token) throw new Error('Sign-in failed: no token');
    localStorage.setItem('era_token', data.access_token);
    location.href = safeNext();
  }

  stepEmail.addEventListener('submit', function (e) {
    e.preventDefault();
    showError('emailError', '');
    var email = (emailInput.value || '').trim().toLowerCase();
    if (!email || email.indexOf('@') < 1) {
      showError('emailError', 'Enter a valid email address');
      return;
    }
    pendingEmail = email;
    showStep('password');
  });

  stepPassword.addEventListener('submit', async function (e) {
    e.preventDefault();
    showError('passwordError', '');
    var btn = stepPassword.querySelector('button[type="submit"]');
    btn.disabled = true;
    try {
      await signIn(pendingEmail, passwordInput.value);
    } catch (err) {
      showError('passwordError', err.message || 'Sign-in failed');
      btn.disabled = false;
    }
  });

  document.getElementById('backEmail').addEventListener('click', function () {
    showStep('email');
  });
  document.getElementById('backFromPassword').addEventListener('click', function () {
    showStep('email');
  });
  document.getElementById('toCreate').addEventListener('click', function () {
    document.getElementById('regEmail').value = emailInput.value || '';
    showStep('create');
  });
  document.getElementById('toSignIn').addEventListener('click', function () {
    showStep('email');
  });

  stepCreate.addEventListener('submit', async function (e) {
    e.preventDefault();
    showError('createError', '');
    var email = (document.getElementById('regEmail').value || '').trim().toLowerCase();
    var pass = document.getElementById('regPassword').value || '';
    var pass2 = document.getElementById('regPassword2').value || '';
    if (!email || email.indexOf('@') < 1) {
      showError('createError', 'Enter a valid email address');
      return;
    }
    if (pass.length < 6) {
      showError('createError', 'Password must be at least 6 characters');
      return;
    }
    if (pass !== pass2) {
      showError('createError', 'Passwords do not match');
      return;
    }
    var btn = stepCreate.querySelector('button[type="submit"]');
    btn.disabled = true;
    try {
      var res = await fetch('/oauth2/staging/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email: email, password: pass }),
      });
      if (!res.ok) {
        var t = await res.text();
        if (res.status === 409) throw new Error('An account with this email already exists');
        if (res.status === 404) throw new Error('Registration is not enabled on this stand');
        throw new Error(t || 'Could not create account (' + res.status + ')');
      }
      await signIn(email, pass);
    } catch (err) {
      showError('createError', err.message || 'Could not create account');
      btn.disabled = false;
    }
  });

  // Already signed in → bounce to next
  if (localStorage.getItem('era_token') && params.get('force') !== '1') {
    location.replace(safeNext());
    return;
  }

  showStep('email');
})();
