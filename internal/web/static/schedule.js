(() => {
  const root = document.querySelector(".table-scroll");
  const table = document.querySelector(".schedule-table");
  if (!root || !table) {
    return;
  }
  const csrf = root.getAttribute("data-csrf") || "";
  let saveTimer = null;

  function formatEffort(value) {
    const number = Number(value);
    if (!number) {
      return "";
    }
    return number.toFixed(1);
  }

  function formatSum(value) {
    const number = Number(value);
    if (Number.isNaN(number)) {
      return "0.0";
    }
    return number.toFixed(1);
  }

  function syncStickyOffsets() {
    const header = table.querySelector("thead tr:first-child");
    const thead = table.querySelector("thead");
    if (!header) {
      return;
    }
    const project = header.querySelector(".col-project");
    const phase = header.querySelector(".col-phase");
    const meta1 = header.querySelector(".col-meta1");
    const meta2 = header.querySelector(".col-meta2");
    const meta3 = header.querySelector(".col-meta3");
    if (!project || !phase || !meta1 || !meta2 || !meta3) {
      return;
    }
    const projectWidth = project.getBoundingClientRect().width;
    const phaseWidth = phase.getBoundingClientRect().width;
    const meta1Width = meta1.getBoundingClientRect().width;
    const meta2Width = meta2.getBoundingClientRect().width;
    const meta3Width = meta3.getBoundingClientRect().width;
    table.style.setProperty("--sticky-left-phase", `${projectWidth}px`);
    table.style.setProperty("--sticky-left-meta1", `${projectWidth + phaseWidth}px`);
    table.style.setProperty("--sticky-left-meta2", `${projectWidth + phaseWidth + meta1Width}px`);
    table.style.setProperty(
      "--sticky-left-meta3",
      `${projectWidth + phaseWidth + meta1Width + meta2Width}px`
    );
    table.style.setProperty(
      "--sticky-cols-width",
      `${projectWidth + phaseWidth + meta1Width + meta2Width + meta3Width}px`
    );
    if (thead) {
      table.style.setProperty("--sticky-thead-height", `${thead.getBoundingClientRect().height}px`);
    }
  }

  function updatePeriodTotals() {
    const totals = {};
    table.querySelectorAll(".effort-input").forEach((input) => {
      const key = `${input.dataset.ym}-${input.dataset.decade}`;
      const value = Number(input.value || 0);
      totals[key] = (totals[key] || 0) + (Number.isNaN(value) ? 0 : value);
    });
    table.querySelectorAll("[data-period-allocated]").forEach((cell) => {
      const key = cell.getAttribute("data-period-allocated");
      cell.textContent = formatSum(totals[key] || 0);
    });
  }

  function applyTaskMeta(taskId, allocated, diff, diffSign) {
    const allocatedCell = table.querySelector(`[data-phase-allocated="${taskId}"]`);
    const diffCell = table.querySelector(`[data-phase-diff="${taskId}"]`);
    if (allocatedCell) {
      allocatedCell.textContent = allocated;
    }
    if (diffCell) {
      diffCell.textContent = diff;
      diffCell.classList.remove("warn", "diff-over");
      if (diffSign > 0) {
        diffCell.classList.add("warn");
      } else if (diffSign < 0) {
        diffCell.classList.add("diff-over");
      }
    }
  }

  function saveAllocation(input) {
    const body = new URLSearchParams();
    body.set("csrf", csrf);
    body.set("task_id", input.dataset.taskId || "");
    body.set("year_month", input.dataset.ym || "");
    body.set("decade", input.dataset.decade || "");
    body.set("effort", input.value || "");
    return fetch("/projects/allocations", {
      method: "POST",
      headers: { "Content-Type": "application/x-www-form-urlencoded" },
      body,
    }).then(async (res) => {
      if (!res.ok) {
        const text = await res.text();
        throw new Error(text || "保存に失敗しました");
      }
      return res.json();
    }).then((data) => {
      input.value = data.effort === "0.0" ? "" : data.effort;
      const cell = input.closest(".phase-cell");
      if (cell) {
        cell.classList.toggle("active", Boolean(input.value));
      }
      applyTaskMeta(data.task_id, data.allocated, data.diff, data.diff_sign);
      updatePeriodTotals();
    }).catch((err) => {
      window.alert(err.message || "保存に失敗しました");
    });
  }

  function queueSave(input) {
    window.clearTimeout(saveTimer);
    saveTimer = window.setTimeout(() => {
      saveAllocation(input);
    }, 400);
  }

  table.querySelectorAll(".effort-input").forEach((input) => {
    input.addEventListener("change", () => {
      window.clearTimeout(saveTimer);
      saveAllocation(input);
    });
    input.addEventListener("input", () => {
      const cell = input.closest(".phase-cell");
      if (cell) {
        cell.classList.toggle("active", Boolean(input.value) && Number(input.value) > 0);
      }
      queueSave(input);
    });
  });

  syncStickyOffsets();
  window.addEventListener("resize", syncStickyOffsets);

  const current = table.querySelector("th.col-current, td.col-current");
  if (current) {
    current.scrollIntoView({ block: "nearest", inline: "center" });
  }
})();
