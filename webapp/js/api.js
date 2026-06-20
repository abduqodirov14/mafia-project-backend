function detectBase() {
  if (window.API_BASE) return window.API_BASE.replace(/\/$/, '');
  const port = '8080';
  if (location.port === port) return '';
  return `${location.protocol}//${location.hostname}:${port}`;
}

const BASE = detectBase();

export async function api(url, opts = {}) {
  const headers = { 'ngrok-skip-browser-warning': 'true' };
  if (opts.body) headers['Content-Type'] = 'application/json';

  const res = await fetch(BASE + url, { ...opts, headers });
  const text = await res.text();
  if (text.trim().startsWith('<')) throw new Error('Server xato');
  return JSON.parse(text);
}

export async function createRoom(userID, username) {
  return api(`/api/room/create?user=${userID}&name=${encodeURIComponent(username)}`);
}

export async function joinRoom(roomID, userID, username) {
  return api('/api/room/join', {
    method: 'POST',
    body: JSON.stringify({ room_id: roomID, user_id: userID, name: username }),
  });
}

export async function getRoomInfo(roomID) {
  return api(`/api/room/info?room=${roomID}`);
}
