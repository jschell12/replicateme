// Quirk slider interaction
const slider = document.getElementById('quirk-slider');
const valueDisplay = document.getElementById('quirk-value');
const examples = document.querySelectorAll('.quirk-example');

const levels = [0, 25, 50, 75, 100];

function getClosestLevel(val) {
  let closest = levels[0];
  let minDist = Math.abs(val - closest);
  for (const level of levels) {
    const dist = Math.abs(val - level);
    if (dist < minDist) {
      minDist = dist;
      closest = level;
    }
  }
  return closest;
}

function updateQuirkDisplay(val) {
  valueDisplay.textContent = val + '%';
  const activeLevel = getClosestLevel(val);
  examples.forEach(ex => {
    const exLevel = parseInt(ex.dataset.level, 10);
    if (exLevel === activeLevel) {
      ex.classList.add('active');
    } else {
      ex.classList.remove('active');
    }
  });
}

if (slider) {
  slider.addEventListener('input', (e) => {
    updateQuirkDisplay(parseInt(e.target.value, 10));
  });
}

// Typing effect for hero terminal (subtle)
function typeTerminal() {
  const heroLine = document.querySelector('.hero-terminal .terminal-line');
  if (!heroLine) return;

  const text = heroLine.textContent;
  const prompt = heroLine.querySelector('.prompt');
  if (!prompt) return;

  // Already has content, just add a blinking cursor
  const cursor = document.createElement('span');
  cursor.className = 'cursor';
  cursor.textContent = '\u258B';
  cursor.style.cssText = 'animation: blink 1s step-end infinite; color: var(--accent); margin-left: 2px;';
  heroLine.appendChild(cursor);

  // Add blink animation
  if (!document.getElementById('cursor-style')) {
    const style = document.createElement('style');
    style.id = 'cursor-style';
    style.textContent = '@keyframes blink { 0%, 100% { opacity: 1; } 50% { opacity: 0; } }';
    document.head.appendChild(style);
  }
}

typeTerminal();
