import './style.css';
import { GetState, Apply, Reset, SelectDisplay } from '../wailsjs/go/main/App';

// Slider definitions, mirroring the AMD Adrenalin custom color panel.
// `nvapi: true` marks controls that need the NVIDIA driver path.
const SLIDERS = [
  { key: 'temperature', label: 'Color temperature', min: 1000, max: 10000, step: 100, unit: 'K', neutral: 6500 },
  { key: 'brightness',  label: 'Brightness',        min: 0,    max: 200,   step: 1,   unit: '%', neutral: 100 },
  { key: 'contrast',    label: 'Contrast',          min: 0,    max: 200,   step: 1,   unit: '%', neutral: 100 },
  { key: 'gamma',       label: 'Gamma',             min: 0.3,  max: 2.8,   step: 0.05, unit: '', neutral: 1.0 },
  { key: 'saturation',  label: 'Saturation',        min: 0,    max: 100,   step: 1,   unit: '%', neutral: 0, enabled: (s) => s.saturationAvailable },
  // Hue bounds come from the driver: NvAPI allows ±180, AMD ADL typically ±30.
  { key: 'hue',         label: 'Hue',               min: (s) => s.hueMin, max: (s) => s.hueMax, step: 1, unit: '°', neutral: 0, enabled: (s) => s.hueAvailable },
];

let settings = {};
let applyTimer = null;

function fmt(def, value) {
  const v = def.step < 1 ? Number(value).toFixed(2) : Math.round(value);
  return `${v}${def.unit}`;
}

function render(state) {
  settings = state.settings;

  const rows = SLIDERS.map((def) => {
    const disabled = def.enabled ? !def.enabled(state) : false;
    const min = typeof def.min === 'function' ? def.min(state) : def.min;
    const max = typeof def.max === 'function' ? def.max(state) : def.max;
    return `
      <div class="row ${disabled ? 'disabled' : ''}">
        <div class="row-head">
          <label for="${def.key}">${def.label}</label>
          <span class="value" id="${def.key}-value">${fmt(def, settings[def.key])}</span>
        </div>
        <input type="range" id="${def.key}" min="${min}" max="${max}"
               step="${def.step}" value="${settings[def.key]}" ${disabled ? 'disabled' : ''} />
      </div>`;
  }).join('');

  // Custom dropdown — the native <select> popup is painted by the OS
  // theme (white on Windows light mode) and cannot be styled.
  const displays = state.displays || [];
  const current = displays.find((d) => d.id === state.selected);
  const items = displays.map((d) =>
    `<button class="select-item ${d.id === state.selected ? 'selected' : ''}" data-id="${d.id}">${d.name}</button>`
  ).join('');

  document.querySelector('#app').innerHTML = `
    <header>
      <h1>Teinte <span class="version">${state.version}</span></h1>
      <div class="head-controls">
        <div class="select" id="display-select">
          <button class="select-btn" id="display-btn" title="Gamma settings are per display">
            <span>${current ? current.name : 'No display'}</span><span class="chevron">▾</span>
          </button>
          <div class="select-panel hidden" id="display-panel">${items}</div>
        </div>
        <button id="reset">Reset</button>
      </div>
    </header>
    <main>${rows}</main>
    <footer>
      <div class="backend">gamma: ${state.gammaBackend}</div>
      <div class="backend ${state.saturationAvailable ? '' : 'off'}">vendor: ${state.vendorBackend}</div>
      ${state.saturationAvailable ? '<div class="backend">saturation &amp; hue are GPU-wide (all its displays)</div>' : ''}
      <div class="errors" id="errors">${state.errors || ''}</div>
    </footer>`;

  for (const def of SLIDERS) {
    const input = document.getElementById(def.key);
    if (!input) continue;
    input.addEventListener('input', () => {
      const value = def.step < 1 ? parseFloat(input.value) : parseInt(input.value, 10);
      settings[def.key] = value;
      document.getElementById(`${def.key}-value`).textContent = fmt(def, value);
      scheduleApply();
    });
  }

  document.getElementById('reset').addEventListener('click', async () => {
    render(await Reset());
  });

  const panel = document.getElementById('display-panel');
  document.getElementById('display-btn').addEventListener('click', (e) => {
    e.stopPropagation();
    panel.classList.toggle('hidden');
  });
  panel.querySelectorAll('.select-item').forEach((item) => {
    item.addEventListener('click', async () => {
      clearTimeout(applyTimer); // pending apply belongs to the previous display
      render(await SelectDisplay(item.dataset.id));
    });
  });
  // Assignment (not addEventListener) so each render replaces the
  // previous handler instead of stacking them.
  document.onclick = () => panel.classList.add('hidden');
  document.onkeydown = (e) => {
    if (e.key === 'Escape') panel.classList.add('hidden');
  };
}

// Debounce so dragging a slider does not spam the driver: last position
// wins, applied at most every 100ms.
function scheduleApply() {
  clearTimeout(applyTimer);
  applyTimer = setTimeout(async () => {
    const state = await Apply(settings);
    document.getElementById('errors').textContent = state.errors || '';
  }, 100);
}

GetState().then(render);
