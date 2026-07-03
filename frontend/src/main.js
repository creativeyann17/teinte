import './style.css';
import { GetState, Apply, Reset, SelectDisplay, ApplyPreset } from '../wailsjs/go/main/App';

// Sliders of the "All" view, mirroring the AMD Adrenalin custom color
// panel. Saturation/hue act GPU-wide, hence the divider before them.
const ALL_SLIDERS = [
  { key: 'temperature', label: 'Color temperature', min: 1000, max: 10000, step: 100, unit: 'K' },
  { key: 'brightness',  label: 'Brightness',        min: 0,    max: 200,   step: 1,   unit: '%' },
  { key: 'contrast',    label: 'Contrast',          min: 0,    max: 200,   step: 1,   unit: '%' },
  { key: 'gamma',       label: 'Gamma',             min: 0.3,  max: 2.8,   step: 0.05, unit: '' },
  { key: 'saturation',  label: 'Saturation',        min: 0,    max: 100,   step: 1,   unit: '%', enabled: (s) => s.saturationAvailable, divider: true },
  // Hue bounds come from the driver: NvAPI allows ±180, AMD ADL typically ±30.
  { key: 'hue',         label: 'Hue',               min: (s) => s.hueMin, max: (s) => s.hueMax, step: 1, unit: '°', enabled: (s) => s.hueAvailable },
];

// Per-channel views (Red/Green/Blue): same gamma-ramp trio, applied on
// top of the global sliders — this is how panel looks like ASUS "Vivid"
// are reproduced (e.g. pulling the blue channel down).
const CHANNEL_SLIDERS = [
  { key: 'brightness', label: 'Brightness', min: 0,   max: 200, step: 1,    unit: '%' },
  { key: 'contrast',   label: 'Contrast',   min: 0,   max: 200, step: 1,    unit: '%' },
  { key: 'gamma',      label: 'Gamma',      min: 0.3, max: 2.8, step: 0.05, unit: '' },
];

const CHANNELS = ['all', 'red', 'green', 'blue'];

let settings = {};
let lastState = null;
let channel = 'all';
let applyTimer = null;

function fmt(def, value) {
  const v = def.step < 1 ? Number(value).toFixed(2) : Math.round(value);
  return `${v}${def.unit}`;
}

const getVal = (def) => channel === 'all' ? settings[def.key] : settings[channel][def.key];
const setVal = (def, v) => {
  if (channel === 'all') settings[def.key] = v;
  else settings[channel][def.key] = v;
  settings.profile = 'Custom'; // manual edit leaves the preset
};

function render(state) {
  lastState = state;
  settings = state.settings;
  const defs = channel === 'all' ? ALL_SLIDERS : CHANNEL_SLIDERS;

  const rows = defs.map((def) => {
    const disabled = def.enabled ? !def.enabled(state) : false;
    const min = typeof def.min === 'function' ? def.min(state) : def.min;
    const max = typeof def.max === 'function' ? def.max(state) : def.max;
    const divider = def.divider ? '<div class="divider">GPU-wide · all displays</div>' : '';
    return `${divider}
      <div class="row ${disabled ? 'disabled' : ''}">
        <div class="row-head">
          <label for="${def.key}">${def.label}</label>
          <span class="value" id="${def.key}-value">${fmt(def, getVal(def))}</span>
        </div>
        <input type="range" id="${def.key}" min="${min}" max="${max}"
               step="${def.step}" value="${getVal(def)}" ${disabled ? 'disabled' : ''} />
      </div>`;
  }).join('');

  const presetBtns = (state.presets || []).map((name) =>
    `<button class="preset-btn ${settings.profile === name ? 'active' : ''}" data-preset="${name}">${name}</button>`
  ).join('');

  const tabs = CHANNELS.map((c) =>
    `<button class="tab ${c} ${channel === c ? 'active' : ''}" data-channel="${c}">${c === 'all' ? 'All' : c[0].toUpperCase()}</button>`
  ).join('');

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
    <div class="body">
      <aside>
        <div class="aside-title">Profiles</div>
        ${presetBtns}
        <button class="preset-btn ${settings.profile === 'Custom' ? 'active' : ''}" disabled>Custom</button>
      </aside>
      <section class="panel">
        <div class="tabs">${tabs}</div>
        <main>${rows}</main>
      </section>
    </div>
    <footer>
      <div class="backend">gamma: ${state.gammaBackend}</div>
      <div class="backend ${state.saturationAvailable ? '' : 'off'}">vendor: ${state.vendorBackend}</div>
      <div class="errors" id="errors">${state.errors || ''}</div>
    </footer>`;

  for (const def of defs) {
    const input = document.getElementById(def.key);
    if (!input) continue;
    input.addEventListener('input', () => {
      const value = def.step < 1 ? parseFloat(input.value) : parseInt(input.value, 10);
      setVal(def, value);
      document.getElementById(`${def.key}-value`).textContent = fmt(def, value);
      document.querySelectorAll('.preset-btn').forEach((b) =>
        b.classList.toggle('active', b.textContent === 'Custom'));
      scheduleApply();
    });
  }

  document.querySelectorAll('.tab').forEach((tab) => {
    tab.addEventListener('click', () => {
      channel = tab.dataset.channel;
      render(lastState);
    });
  });

  document.querySelectorAll('.preset-btn[data-preset]').forEach((btn) => {
    btn.addEventListener('click', async () => {
      clearTimeout(applyTimer);
      render(await ApplyPreset(btn.dataset.preset));
    });
  });

  document.getElementById('reset').addEventListener('click', async () => {
    clearTimeout(applyTimer);
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
