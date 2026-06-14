// Properties panel (§6.11, FR-020a, FR-020b): shows the selected component
// instance's type data (read-only) and lets the user override its propagation
// delays and declared properties for that instance only (FR-058). Edits
// dispatch setOverrideCmd through the store; the panel re-renders on every
// store notification.

import { setOverrideCmd } from "../commands.js";

function el(tag, className, text) {
  const e = document.createElement(tag);
  if (className) e.className = className;
  if (text != null) e.textContent = text;
  return e;
}

function infoRow(label, value) {
  const row = el("div", "prop-row");
  row.append(el("span", "prop-label", label), el("span", "prop-value", value));
  return row;
}

// docSection builds the read-only Documentation block (FR-095) from a type's
// documentation fields (FR-094), or returns null when the type carries none.
// The one-line description and datasheet link sit at the top; the per-pin roles
// live in a collapsed <details> so they never crowd the editable override fields.
function docSection(td) {
  const frag = document.createDocumentFragment();
  let any = false;

  if (td.description) {
    frag.appendChild(el("div", "prop-doc-desc", td.description));
    any = true;
  }

  const ds = td.datasheet;
  if (ds && ds.url) {
    const link = el("a", "prop-doc-link", ds.title || ds.vendor || "Datasheet");
    link.href = ds.url;
    link.target = "_blank";
    link.rel = "noopener";
    if (ds.vendor && ds.title) link.title = `${ds.vendor}: ${ds.title}`;
    const row = el("div", "prop-doc-row");
    row.appendChild(link);
    if (ds.rev) row.appendChild(el("span", "prop-doc-rev", ds.rev));
    frag.appendChild(row);
    any = true;
  }

  // Per-pin roles in a collapsed disclosure (the list can be long).
  const roles = (td.pins ?? []).filter((p) => p.desc);
  if (roles.length > 0) {
    const details = el("details", "prop-doc-details");
    details.appendChild(el("summary", null, "Pin roles"));
    const list = el("dl", "prop-doc-pins");
    for (const p of roles) {
      list.append(el("dt", null, p.name), el("dd", null, p.desc));
    }
    details.appendChild(list);
    frag.appendChild(details);
    any = true;
  }

  return any ? frag : null;
}

// initProperties wires the panel to the store. Returns nothing; lives for the
// app's lifetime.
export function initProperties({ container, store }) {
  function render() {
    container.replaceChildren();
    const sel = store.state.selection;
    const inst =
      sel && sel.kind === "component"
        ? store.design.components.find((c) => c.refdes === sel.refdes)
        : null;
    if (!inst) {
      container.appendChild(el("div", "prop-empty", "No component selected"));
      return;
    }

    const td = inst.typeData;
    container.appendChild(el("div", "prop-title", inst.refdes));
    container.append(
      infoRow("Type", inst.type),
      infoRow("Size", `${td.width} × ${td.height}`),
      infoRow("Pins", String(td.pins.length)),
    );

    // Read-only documentation (FR-094, FR-095): always shown, never locked while
    // simulating since it edits nothing.
    const doc = docSection(td);
    if (doc) container.appendChild(doc);

    // The panel is read-only while a simulation runs (FR-087).
    const locked = store.state.simulating;

    // overrideRow builds one editable numeric field whose value shadows the
    // type default via inst.overrides[group][key] (FR-020a/FR-020b/FR-058).
    function overrideRow(group, key, label, def, unit) {
      const ov = inst.overrides?.[group]?.[key];
      const overridden = ov != null;

      const row = el("div", "prop-row" + (overridden ? " overridden" : ""));
      row.appendChild(el("label", "prop-label", label));

      const input = el("input", "prop-input");
      input.type = "number";
      input.value = String(overridden ? ov : def);
      input.title = `type default: ${def} ${unit}`;
      input.disabled = locked;
      input.addEventListener("change", () => {
        const n = parseFloat(input.value);
        if (!Number.isFinite(n)) {
          render(); // reject invalid input, restore display
          return;
        }
        // Setting back to the type default clears the override.
        store.dispatch(setOverrideCmd(inst.refdes, group, key, n === def ? null : n));
      });
      row.appendChild(input);

      const reset = el("button", "prop-reset", "↺");
      reset.title = "Reset to type default";
      reset.disabled = !overridden || locked;
      reset.addEventListener("click", () =>
        store.dispatch(setOverrideCmd(inst.refdes, group, key, null)),
      );
      row.appendChild(reset);

      return row;
    }

    container.appendChild(el("div", "prop-section", "Propagation delays (ns)"));
    const delays = td.delays ?? {};
    const keys = Object.keys(delays);
    if (keys.length === 0) {
      container.appendChild(el("div", "prop-empty", "none defined for this type"));
    }
    for (const key of keys) {
      container.appendChild(overrideRow("delays", key, key, delays[key], "ns"));
    }

    // Declared properties (FR-020b), e.g. the clock's period/speed (FR-071a).
    const props = td.properties ?? [];
    if (props.length > 0) {
      container.appendChild(el("div", "prop-section", "Properties"));
      for (const p of props) {
        container.appendChild(
          overrideRow("props", p.name, `${p.name} (${p.unit})`, p.default, p.unit),
        );
      }
    }
  }

  store.subscribe(render);
  render();
}
