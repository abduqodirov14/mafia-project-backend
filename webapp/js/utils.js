export function sanitize(s) {
  return String(s || '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

export function fmt(s) {
  return s <= 0 ? '0:00' : `${Math.floor(s / 60)}:${String(s % 60).padStart(2, '0')}`;
}

export function $(id) {
  return document.getElementById(id);
}

export function show(id) {
  document.querySelectorAll('.scr').forEach(s => s.classList.remove('on'));
  $(id)?.classList.add('on');
}

export function currentScreen() {
  return document.querySelector('.scr.on')?.id;
}
