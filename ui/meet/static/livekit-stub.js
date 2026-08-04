// ERA Meet air-gap LiveKit client stub (no CDN). Real deployments vendor livekit-client.js here.
window.ERALiveKit = {
  connect: function (url, token, roomId, statusEl, mode) {
    mode = mode || "stub";
    var line =
      "ERALiveKit mode=" +
      mode +
      " url=" +
      (url || "") +
      " room=" +
      (roomId || "") +
      " token_len=" +
      ((token || "").length);
    if (statusEl) {
      statusEl.textContent =
        (statusEl.textContent || "") +
        "\n" +
        line +
        (mode === "stub" ? "\nSTUB_OK (not media)" : "\nJOIN_OK");
    }
    console.log(line);
    return Promise.resolve({ room: roomId, stub: mode === "stub", mode: mode });
  },
};
