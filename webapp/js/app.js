import { $, show, currentScreen, fmt, sanitize } from './utils.js';
import { api, createRoom, joinRoom, getRoomInfo } from './api.js';
import { GameSocket } from './ws.js';
import { VoiceChat } from './voice.js';
import { initParticles, confetti } from './particles.js';
import {
  renderLobbyTable, renderLobbyList, renderGameTable,
  renderPlayersList, updateHUD, renderActions,
} from './ui.js';

const TgApp = window.Telegram?.WebApp;
if (TgApp) { TgApp.ready(); TgApp.expand(); }

const S = {
  roomID: '',
  userID: 0,
  username: '',
  isOwner: false,
  phase: 'waiting',
  round: 0,
  players: [],
  myRole: '',
  myRoleEmoji: '',
  timer: 0,
  timerT: null,
  selectedAct: null,
  invite: '',
};

let sock, voice;

function getUser() {
  if (TgApp?.initDataUnsafe?.user) {
    const u = TgApp.initDataUnsafe.user;
    return { id: u.id, name: u.username || u.first_name || 'O\'yinchi' };
  }
  let id = localStorage.getItem('muid');
  let name = localStorage.getItem('mnm');
  if (!id) { id = String(Math.floor(Math.random() * 9e5 + 1e5)); localStorage.setItem('muid', id); }
  if (!name) { name = 'Mehmon' + id.slice(-3); localStorage.setItem('mnm', name); }
  return { id: parseInt(id), name };
}

function toast(msg, dur = 2800) {
  const el = $('toast');
  el.textContent = msg;
  el.classList.add('on');
  clearTimeout(el._t);
  el._t = setTimeout(() => el.classList.remove('on'), dur);
}

function startTimer(secs) {
  clearInterval(S.timerT);
  S.timer = secs;
  const el = $('g-tm');
  el.textContent = fmt(secs);
  el.classList.remove('urg');
  S.timerT = setInterval(() => {
    S.timer--;
    el.textContent = fmt(S.timer);
    if (S.timer <= 10) el.classList.add('urg');
    if (S.timer <= 0) clearInterval(S.timerT);
  }, 1000);
}

function addChatMsg(name, text, isMe = false) {
  const c = $('chat-msgs');
  const div = document.createElement('div');
  div.className = 'cmsg';
  const now = new Date();
  const ts = `${now.getHours()}:${String(now.getMinutes()).padStart(2, '0')}`;
  div.innerHTML = `<div class="cm"><span class="cw" style="${isMe ? 'color:#3dba6a' : ''}">${sanitize(name)}</span><span class="ct">${ts}</span></div><div class="ctxt">${text}</div>`;
  c.appendChild(div);
  setTimeout(() => c.scrollTop = c.scrollHeight, 50);
}

function addSysMsg(text) {
  if (!text) return;
  const c = $('chat-msgs');
  const div = document.createElement('div');
  div.className = 'cmsg sys';
  div.innerHTML = `<div class="ctxt">${sanitize(text)}</div>`;
  c.appendChild(div);
  setTimeout(() => c.scrollTop = c.scrollHeight, 50);
}

function connectWS() {
  if (!S.roomID) return;
  sock = new GameSocket(S.roomID, S.userID);
  voice = new VoiceChat(S.userID, (t, p) => sock.send(t, p));

  sock.on('connected', () => addSysMsg('✅ Ulandi'));
  sock.on('disconnected', () => addSysMsg('⚠️ Uzildi...'));

  sock.on('room_info', onRoomInfo);
  sock.on('role_reveal', onRoleReveal);
  sock.on('game_state', onGameState);
  sock.on('phase_change', onPhaseChange);
  sock.on('player_died', onPlayerDied);
  sock.on('game_end', onGameEnd);
  sock.on('chat', onChat);
  sock.on('voice_signal', d => voice.handleSignal(d));
  sock.on('sheriff_result', d => addSysMsg('🕵️ ' + d.result));
  sock.on('night_result', d => addSysMsg(d.result));

  sock.connect();
}

function onRoomInfo(d) {
  if (d.players) S.players = d.players;
  $('lb-id').textContent = d.room_id || S.roomID;
  const cnt = d.count || S.players.length;
  $('lb-cnt').innerHTML = `${cnt}<span class="id-max">/${d.max || 15}</span>`;
  if (d.owner_id) S.isOwner = S.userID === d.owner_id;
  if (d.invite_link) S.invite = d.invite_link;

  const invText = S.invite || location.origin + '/webapp/?room=' + (d.room_id || S.roomID);
  $('lb-inv').textContent = invText;
  S.invite = invText;

  renderLobbyTable(S.players);
  renderLobbyList(S.players, S.userID);

  const startBtn = $('start-btn');
  const waitT = $('wait-t');
  const minInfo = $('min-info');

  if (S.isOwner) {
    startBtn.disabled = cnt < 4;
    startBtn.style.display = 'block';
    waitT.style.display = 'none';
    minInfo.textContent = cnt < 4 ? `Kamida 4 o'yinchi kerak (${cnt}/4)` : '';
  } else {
    startBtn.style.display = 'none';
    waitT.style.display = 'block';
    minInfo.textContent = '';
  }

  if (currentScreen() === 's0') show('s2');
}

function onRoleReveal(d) {
  $('r-em').textContent = d.emoji || '🎭';
  $('r-nm').textContent = (d.role || '').toUpperCase();
  $('r-ds').textContent = d.description || '';
  S.myRole = d.role || '';
  S.myRoleEmoji = d.emoji || '';
  show('s3');
}

function onGameState(d) {
  S.phase = d.phase;
  S.round = d.round;
  if (d.players) S.players = d.players;
  renderGameTable(S.players, S.userID);
  renderPlayersList(S.players, S.userID);
  updateHUD(S.phase, S.round);
  renderActions(S.phase, S.myRole, S.players, S.userID, S.selectedAct);
  if (currentScreen() === 's2' || currentScreen() === 's3') show('s4');
}

function onPhaseChange(d) {
  S.phase = d.phase;
  S.round = d.round;
  S.selectedAct = null;
  updateHUD(S.phase, S.round);
  startTimer(d.timer || 60);
  if (d.message) addSysMsg(d.message);
  renderActions(S.phase, S.myRole, S.players, S.userID, null);
  if (currentScreen() !== 's4') show('s4');
}

function onPlayerDied(d) {
  const p = S.players.find(x => x.id === d.player_id);
  if (p) { p.is_alive = false; p.role = d.role; }
  renderGameTable(S.players, S.userID);
  renderPlayersList(S.players, S.userID);
  const txt = d.voted_out ? `⚖️ ${d.name} chiqarildi (${d.role})` : `💀 ${d.name} o'ldirildi`;
  addSysMsg(txt);
  toast(txt);
}

function onGameEnd(d) {
  clearInterval(S.timerT);
  const icons = { town: '🎉', mafia: '😈', manyak: '🔪', suidsid: '🧌' };
  $('e-em').textContent = icons[d.winner] || '🏆';
  $('e-tt').textContent = d.title || 'O\'YIN TUGADI';
  $('e-sb').textContent = '';
  show('s5');
  if (d.winner === 'town') confetti();
}

function onChat(d) {
  addChatMsg(d.username, sanitize(d.text), d.user_id === S.userID);
}

async function autoJoin() {
  try {
    const info = await getRoomInfo(S.roomID);
    if (info.ok) {
      const j = await joinRoom(S.roomID, S.userID, S.username);
      if (j.ok || j.error?.includes('allaqachon')) {
        connectWS();
        show('s2');
      } else {
        toast('❌ ' + j.error);
        setTimeout(() => show('s1'), 2000);
      }
    } else {
      toast('❌ Xona topilmadi');
      setTimeout(() => show('s1'), 2000);
    }
  } catch (e) {
    toast('❌ ' + e.message);
    setTimeout(() => show('s1'), 2500);
  }
}

async function doCreate() {
  toast('⏳ Xona yaratilmoqda...');
  try {
    const d = await createRoom(S.userID, S.username);
    if (!d.ok) { toast('❌ ' + (d.error || 'Xato')); return; }
    S.roomID = d.room_id;
    S.isOwner = true;
    S.invite = d.invite_link || location.origin + '/webapp/?room=' + d.room_id;
    history.replaceState({}, '', location.pathname + '?room=' + S.roomID);
    connectWS();
    $('lb-id').textContent = S.roomID;
    $('lb-inv').textContent = S.invite;
    show('s2');
  } catch (e) {
    toast('❌ ' + e.message);
  }
}

async function doJoin() {
  const rid = $('rid').value.trim();
  if (!rid) { toast('❌ Xona ID kiriting!'); return; }
  toast('⏳ Qo\'shilmoqda...');
  try {
    const d = await joinRoom(rid, S.userID, S.username);
    if (!d.ok && !d.error?.includes('allaqachon')) { toast('❌ ' + (d.error || 'Xato')); return; }
    S.roomID = rid;
    history.replaceState({}, '', location.pathname + '?room=' + rid);
    connectWS();
    $('lb-id').textContent = rid;
    $('lb-inv').textContent = location.origin + '/webapp/?room=' + rid;
    show('s2');
  } catch (e) {
    toast('❌ ' + e.message);
  }
}

function doAct(targetID, targetName) {
  S.selectedAct = targetID;
  renderActions(S.phase, S.myRole, S.players, S.userID, S.selectedAct);
  if (S.phase === 'voting') {
    sock.send('day_vote', { target_id: targetID });
    toast('✅ ' + targetName + ' ga ovoz berildi');
  } else {
    sock.send('night_action', { role: S.myRole, target_id: targetID });
    toast('✅ ' + targetName + ' tanlandi');
  }
}

async function doDemo() {
  toast('🤖 Demo o\'yin yaratilmoqda...');
  try {
    const d = await api(`/api/demo/start?user=${S.userID}&name=${encodeURIComponent(S.username)}`);
    if (!d.ok) { toast('❌ ' + (d.error || 'Xato')); return; }
    S.roomID = d.room_id;
    S.isOwner = true;
    S.invite = d.invite_link || '';
    history.replaceState({}, '', location.pathname + '?room=' + S.roomID);
    connectWS();
    show('s2');
  } catch (e) {
    toast('❌ ' + e.message);
  }
}

function bindEvents() {
  $('btn-create').addEventListener('click', doCreate);
  $('btn-demo').addEventListener('click', doDemo);
  $('btn-join-input').addEventListener('click', doJoin);
  $('rid').addEventListener('keydown', e => { if (e.key === 'Enter') doJoin(); });
  $('start-btn').addEventListener('click', () => { sock.send('start_game', {}); toast('⏳ O\'yin boshlanmoqda...'); });
  $('btn-role-cont').addEventListener('click', () => show('s4'));
  $('btn-new-game').addEventListener('click', () => { location.href = location.pathname; });

  $('lb-id').addEventListener('click', () => {
    navigator.clipboard?.writeText(S.roomID).then(() => toast('✅ ID nusxa olindi!'));
    TgApp?.HapticFeedback?.impactOccurred('light');
  });
  $('btn-copy-inv').addEventListener('click', () => {
    navigator.clipboard?.writeText(S.invite).then(() => toast('✅ Havola nusxa olindi!'));
    TgApp?.HapticFeedback?.impactOccurred('light');
  });

  // Chat
  $('btn-ch').addEventListener('click', () => {
    $('ch-pn').classList.add('on'); $('ch-bd').classList.add('on'); $('btn-ch').classList.add('on');
    setTimeout(() => { $('ci').focus(); }, 300);
  });
  function closeChat() {
    $('ch-pn').classList.remove('on'); $('ch-bd').classList.remove('on'); $('btn-ch').classList.remove('on');
  }
  $('ch-bd').addEventListener('click', closeChat);
  $('ch-close').addEventListener('click', closeChat);
  $('btn-send-chat').addEventListener('click', sendChatMsg);
  $('ci').addEventListener('keydown', e => { if (e.key === 'Enter') sendChatMsg(); });

  function sendChatMsg() {
    const inp = $('ci');
    const txt = inp.value.trim();
    if (!txt) return;
    sock.send('chat', { user_id: S.userID, username: S.username, text: txt });
    addChatMsg(S.username, sanitize(txt), true);
    inp.value = '';
  }

  // Players panel
  $('btn-pp').addEventListener('click', () => {
    renderPlayersList(S.players, S.userID);
    $('pp-pn').classList.add('on'); $('pp-bd').classList.add('on'); $('btn-pp').classList.add('on');
  });
  function closePP() { $('pp-pn').classList.remove('on'); $('pp-bd').classList.remove('on'); $('btn-pp').classList.remove('on'); }
  $('pp-bd').addEventListener('click', closePP);
  $('pp-close').addEventListener('click', closePP);

  // Voice
  $('mic-btn').addEventListener('click', async () => {
    try {
      await voice.toggle();
      if (voice.active) {
        $('mic-btn').classList.add('live');
        $('mic-l').textContent = 'Gapiryapsiz';
        toast('🎤 Yoqildi');
        voice.connectToPeers(S.players);
      } else {
        $('mic-btn').classList.remove('live');
        $('mic-l').textContent = 'Ovoz';
      }
    } catch (e) {
      toast('❌ ' + e.message);
    }
  });

  // Action buttons delegation
  $('act-grid').addEventListener('click', (e) => {
    const btn = e.target.closest('.act-btn');
    if (!btn) return;
    const id = parseInt(btn.dataset.id);
    const player = S.players.find(p => p.id === id);
    if (player) doAct(id, player.name);
  });
}

// INIT
initParticles();
const user = getUser();
S.userID = user.id;
S.username = user.name;
const q = new URLSearchParams(location.search);
S.roomID = q.get('room') || '';
bindEvents();
if (S.roomID) autoJoin();
else setTimeout(() => show('s1'), 1800);
