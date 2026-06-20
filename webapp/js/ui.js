import { sanitize } from './utils.js';
import { $ } from './utils.js';

const PHASE_META = {
  night: { icon: '🌙', label: 'TUN' },
  day: { icon: '☀️', label: 'KUNDUZ' },
  voting: { icon: '🗳️', label: 'OVOZ BERISH' },
  waiting: { icon: '⏳', label: 'KUTISH' },
};

const NIGHT_ROLES = {
  Mafiya: '😈 KIMNI O\'LDIRASIZ?',
  Don: '😈 KIMNI O\'LDIRASIZ?',
  Shifokor: '👨‍⚕️ KIMNI DAVOLAYSIZ?',
  Komissar: '🕵️ KIMNI TEKSHIRASIZ?',
  Serjant: '👮 KIMNI KUZATASIZ?',
  Mashuqa: '💃 KIMNI BLOKLAYSIZ?',
  Daydi: '🧙 KIM UYIDA TUNAYSIZ?',
  Manyak: '🔪 KIMNI O\'LDIRASIZ?',
  Tentak: '👨🏻‍🦲 KIMG A BORASIZ?',
};

export function renderLobbyTable(players, containerID = 'lb-toks', emptyID = 'lb-em') {
  const container = $(containerID);
  const empty = $(emptyID);
  container.querySelectorAll('.lb-tok').forEach(t => t.remove());

  if (!players.length) {
    empty.style.display = 'flex';
    return;
  }
  empty.style.display = 'none';

  const cx = 102, cy = 102, radius = 76;
  players.forEach((p, i) => {
    const angle = (i / players.length) * 2 * Math.PI - Math.PI / 2;
    const x = cx + radius * Math.cos(angle);
    const y = cy + radius * Math.sin(angle);
    const isYou = p.id === State.userID;

    const el = document.createElement('div');
    el.className = 'lb-tok';
    el.style.cssText = `left:${x}px;top:${y}px;animation-delay:${i * 130}ms`;
    el.innerHTML = `<div class="lb-pawn ${isYou ? 'you' : 'oth'}"></div><div class="lb-pn ${isYou ? 'you' : ''}">${isYou ? '⭐' : ''}${sanitize(p.name)}</div>`;
    container.appendChild(el);
  });
}

export function renderLobbyList(players, myID) {
  $('pl-list').innerHTML = players.map(p => `
    <div class="pl-item">
      <div class="pl-av ${p.id === myID ? 'you' : ''}">${(p.name || '?')[0].toUpperCase()}</div>
      <div class="pl-nm">${sanitize(p.name)}</div>
      ${p.join_order === 1 ? '<div class="pl-bg ow">👑 EGA</div>' : ''}
      ${p.id === myID ? '<div class="pl-bg yu">SIZ</div>' : ''}
    </div>
  `).join('');
}

export function renderGameTable(players, myID) {
  const container = $('toks');
  container.innerHTML = '';
  const n = players.length;
  if (!n) return;

  const cx = 122, cy = 122, radius = 92;
  players.forEach((p, i) => {
    const angle = (i / n) * 2 * Math.PI - Math.PI / 2;
    const x = cx + radius * Math.cos(angle);
    const y = cy + radius * Math.sin(angle);
    const isYou = p.id === myID;
    const dead = !p.is_alive;

    const el = document.createElement('div');
    el.className = `tok ${dead ? 'dead' : isYou ? 'you' : 'alive'}`;
    el.style.cssText = `left:${x}px;top:${y}px`;
    el.innerHTML = `<div class="tok-b"></div><div class="tok-lb">${dead ? '💀' : ''}${isYou ? '⭐' : ''}${sanitize(p.name)}</div>`;
    container.appendChild(el);
  });

  const ctr = document.createElement('div');
  ctr.className = 'tok ctr';
  ctr.style.cssText = `left:${cx}px;top:${cy}px`;
  ctr.innerHTML = '<div class="tok-b"></div>';
  container.appendChild(ctr);
}

export function renderPlayersList(players, myID) {
  $('pp-list').innerHTML = players.map(p => `
    <div class="pp-row ${!p.is_alive ? 'dead' : ''}">
      <div class="pp-av">${(p.name || '?')[0].toUpperCase()}</div>
      <div class="pp-nm">${sanitize(p.name)}${p.id === myID ? ' (Siz)' : ''}</div>
      <div>${p.is_alive ? '🟢' : '💀'}</div>
    </div>
  `).join('');
}

export function updateHUD(phase, round) {
  const meta = PHASE_META[phase] || PHASE_META.waiting;
  $('ph-ic').textContent = meta.icon;
  $('ph-nm').textContent = meta.label;
  $('g-rn').textContent = 'TUR ' + round;
}

export function renderActions(phase, myRole, players, myID, selectedID) {
  const sec = $('act-sec');
  const grid = $('act-grid');
  const lbl = $('act-lbl');

  const isNight = phase === 'night';
  const isVoting = phase === 'voting';

  if ((!isNight && !isVoting) || !myRole) {
    sec.style.display = 'none';
    return;
  }

  const alive = players.filter(p => p.is_alive && p.id !== myID);

  if (isNight && !NIGHT_ROLES[myRole]) {
    sec.style.display = 'none';
    return;
  }

  lbl.textContent = isVoting ? '🗳 KIMNI CHIQARAMIZ?' : NIGHT_ROLES[myRole];
  grid.innerHTML = alive.map(p => `
    <div class="act-btn ${selectedID === p.id ? 'sel' : ''}" data-id="${p.id}">
      <span class="act-ic">${isVoting ? '👈' : '🎯'}</span>
      <span class="act-nm">${sanitize(p.name)}</span>
    </div>
  `).join('');
  sec.style.display = 'block';
}
