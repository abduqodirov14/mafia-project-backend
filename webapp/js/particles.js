export function initParticles() {
  const canvas = document.getElementById('fc');
  const ctx = canvas.getContext('2d');
  let W, H;
  let particles = [];

  function resize() {
    W = canvas.width = innerWidth;
    H = canvas.height = innerHeight;
  }

  function spawn() {
    return {
      x: Math.random() * W,
      y: H + 8,
      vx: (Math.random() - 0.5) * 0.7,
      vy: -(Math.random() * 1.2 + 0.5),
      life: 1,
      decay: Math.random() * 0.004 + 0.003,
      r: Math.random() * 2.5 + 1,
      hue: Math.random() * 30 + 5,
    };
  }

  function loop() {
    ctx.clearRect(0, 0, W, H);
    for (let i = 0; i < 3; i++) {
      if (particles.length < 100) particles.push(spawn());
    }
    particles = particles.filter(p => p.life > 0);
    for (const p of particles) {
      p.x += p.vx;
      p.y += p.vy;
      p.life -= p.decay;
      p.vx += (Math.random() - 0.5) * 0.05;
      ctx.globalAlpha = p.life * 0.4;
      ctx.fillStyle = `hsl(${p.hue},90%,${50 + p.life * 20}%)`;
      ctx.beginPath();
      ctx.arc(p.x, p.y, p.r, 0, Math.PI * 2);
      ctx.fill();
    }
    ctx.globalAlpha = 1;
    requestAnimationFrame(loop);
  }

  resize();
  window.addEventListener('resize', resize);
  loop();
}

export function confetti() {
  const cv = document.createElement('canvas');
  cv.className = 'cvc';
  cv.width = innerWidth;
  cv.height = innerHeight;
  document.body.appendChild(cv);
  const ctx = cv.getContext('2d');
  const colors = ['#c8a55e', '#27ae60', '#2980b9', '#e74c3c', '#f1c40f'];
  const dots = Array.from({ length: 120 }, () => ({
    x: Math.random() * innerWidth,
    y: Math.random() * -300,
    r: Math.random() * 5 + 3,
    c: colors[Math.floor(Math.random() * 5)],
    vx: (Math.random() - 0.5) * 2.5,
    vy: Math.random() * 3.5 + 1.5,
  }));

  let frame = 0;
  function draw() {
    ctx.clearRect(0, 0, cv.width, cv.height);
    for (const d of dots) {
      ctx.fillStyle = d.c;
      ctx.beginPath();
      ctx.arc(d.x, d.y, d.r, 0, Math.PI * 2);
      ctx.fill();
      d.x += d.vx;
      d.y += d.vy;
    }
    if (++frame < 200) requestAnimationFrame(draw);
    else cv.remove();
  }
  draw();
}
