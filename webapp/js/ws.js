function detectBase() {
  if (window.API_BASE) return window.API_BASE.replace(/\/$/, '');
  const port = '8080';
  if (location.port === port) return '';
  return `${location.protocol}//${location.hostname}:${port}`;
}

function getWSBase() {
  const base = detectBase();
  if (base) {
    return base.replace('https://', 'wss://').replace('http://', 'ws://');
  }
  return `${location.protocol === 'https:' ? 'wss' : 'ws'}://${location.host}`;
}

export class GameSocket {
  constructor(roomID, userID) {
    this.roomID = roomID;
    this.userID = userID;
    this.ws = null;
    this.handlers = {};
  }

  on(type, handler) {
    this.handlers[type] = handler;
  }

  connect() {
    const url = `${getWSBase()}/ws?room=${this.roomID}&user=${this.userID}`;
    this.ws = new WebSocket(url);

    this.ws.onopen = () => this._emit('connected');
    this.ws.onmessage = (e) => {
      try {
        const msg = JSON.parse(e.data);
        let payload = {};
        try { payload = msg.payload ? JSON.parse(msg.payload) : {}; } catch {}
        this._emit(msg.type, payload, msg);
      } catch {}
    };
    this.ws.onclose = () => {
      this._emit('disconnected');
      setTimeout(() => this.connect(), 3000);
    };
  }

  send(type, payload = {}) {
    if (this.ws?.readyState === 1) {
      this.ws.send(JSON.stringify({
        type,
        room_id: this.roomID,
        user_id: this.userID,
        payload: JSON.stringify(payload),
      }));
    }
  }

  _emit(type, ...args) {
    this.handlers[type]?.(...args);
  }
}
