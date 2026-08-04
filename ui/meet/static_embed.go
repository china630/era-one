package meet

import (
	"embed"
	"io/fs"
)

//go:embed static/*
var embeddedStatic embed.FS

func staticFS() fs.FS {
	sub, err := fs.Sub(embeddedStatic, "static")
	if err != nil {
		return embeddedStatic
	}
	return sub
}

// joinHTML — air-gap join page; LiveKit client from /meet/static/livekit-stub.js (no CDN).
const joinHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8"/>
<title>ERA Meet</title>
<style>
body{font-family:system-ui,sans-serif;margin:2rem;background:#0f1419;color:#e7ecf1}
input,button{padding:.5rem .75rem;margin:.25rem 0}
#status{margin-top:1rem;opacity:.85}
</style>
</head>
<body>
<h1>ERA Meet</h1>
<p>Join via room API (air-gap LiveKit stub — no CDN).</p>
<label>Room name <input id="name" value="era-meet"/></label><br/>
<label>Identity <input id="identity" value="lab-user"/></label><br/>
<button id="join">Join</button>
<pre id="status"></pre>
<script src="/meet/static/livekit-stub.js"></script>
<script>
async function join(){
  const name=document.getElementById('name').value;
  const identity=document.getElementById('identity').value;
  const st=document.getElementById('status');
  st.textContent='requesting room…';
  const r=await fetch('/meet/api/room?name='+encodeURIComponent(name)+'&identity='+encodeURIComponent(identity),{headers:{'X-ERA-Tenant':'t-demo','X-ERA-Role':'vcs.user','X-ERA-User':identity}});
  if(!r.ok){st.textContent='error '+r.status;return;}
  const j=await r.json();
  st.textContent=JSON.stringify(j,null,2);
  if(window.ERALiveKit&&window.ERALiveKit.connect){
    window.ERALiveKit.connect(j.livekit_url,j.token,j.room_id,st,j.mode);
  }
}
document.getElementById('join').onclick=join;
</script>
</body>
</html>
`
