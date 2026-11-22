(() => {
  const root = document.querySelector(".table-scroll");
  const table = document.querySelector(".schedule-table");
  if (!root || !table) {
    return;
  }
  const csrf = root.getAttribute("data-csrf") || "";
  const actualModeButton = document.getElementById("actual-mode-button");
  const inputs = table.querySelectorAll(".effort-input");
  let actualMode = false;
  let saveTimer = null;
  let activeRow = null;
  let activeColEls = [];

  function formatSum(value) {
    const number = Number(value);
    if (Number.isNaN(number)) {
      return "0.0";
    }
    return number.toFixed(1);
  }

  function syncEffortStyle(fieldOrInput) {
    const field = fieldOrInput.classList?.contains("effort-field")
      ? fieldOrInput
      : fieldOrInput.closest(".effort-field");
    const cell = field?.closest(".phase-cell");
    if (!field || !cell) {
      return;
    }
    const input = field.querySelector(".effort-input");
    const plan = field.querySelector(".effort-plan");
    const display = field.querySelector(".effort-display");
    const label = field.querySelector(".effort-float-label");
    const estimate = input?.dataset.estimate || plan?.textContent || "";
    const actual = input?.dataset.actual || display?.textContent || "";
    cell.classList.toggle("active", cellIsActive(cell, estimate && Number(estimate) > 0));
    cell.classList.toggle("has-actual", Number(actual || 0) > 0);
    if (label) {
      label.textContent = estimate;
    }
    if (display && input) {
      display.textContent = actual;
    }
  }

  function applyFieldForMode(field) {
    const input = field.querySelector(".effort-input");
    if (input) {
      const estimate = input.dataset.estimate || "";
      input.value = estimate;
      input.disabled = actualMode;
      if (actualMode && document.activeElement === input) {
        input.blur();
      }
    }
    syncEffortStyle(field);
  }

  // 残工数行を隠すと rowspan が余るので、実績モードでは工程・案件の行数を詰める。
  function adjustActualModeRowspans(enabled) {
    table.querySelectorAll("tbody.project-block").forEach((block) => {
      const remainCount = block.querySelectorAll("tr.phase-capacity").length;
      if (remainCount === 0) {
        return;
      }
      const project = block.querySelector("th.col-project[rowspan]");
      if (project) {
        if (enabled) {
          if (!project.dataset.fullRowspan) {
            project.dataset.fullRowspan = project.getAttribute("rowspan") || "1";
          }
          const full = Number(project.dataset.fullRowspan) || 1;
          project.setAttribute("rowspan", String(Math.max(1, full - remainCount)));
        } else if (project.dataset.fullRowspan) {
          project.setAttribute("rowspan", project.dataset.fullRowspan);
          delete project.dataset.fullRowspan;
        }
      }
      block.querySelectorAll("td.col-phase[rowspan]").forEach((cell) => {
        if (enabled) {
          if (!cell.dataset.fullRowspan) {
            cell.dataset.fullRowspan = cell.getAttribute("rowspan") || "1";
          }
          const full = Number(cell.dataset.fullRowspan) || 1;
          cell.setAttribute("rowspan", String(Math.max(1, full - 1)));
        } else if (cell.dataset.fullRowspan) {
          cell.setAttribute("rowspan", cell.dataset.fullRowspan);
          delete cell.dataset.fullRowspan;
        }
      });
    });
  }

  function setActualMode(enabled) {
    actualMode = enabled;
    table.classList.toggle("actual-mode", enabled);
    if (actualModeButton) {
      actualModeButton.setAttribute("aria-pressed", enabled ? "true" : "false");
      actualModeButton.title = enabled ? "割当表示モードに戻る" : "実績表示モードにする";
    }
    if (enabled) {
      clearActiveRow();
    }
    adjustActualModeRowspans(enabled);
    table.querySelectorAll(".effort-field").forEach((field) => {
      applyFieldForMode(field);
    });
    table.querySelectorAll(".sum-single").forEach((el) => {
      el.setAttribute("aria-hidden", enabled ? "true" : "false");
    });
    table.querySelectorAll(".sum-fraction").forEach((el) => {
      el.setAttribute("aria-hidden", enabled ? "false" : "true");
    });
    syncStickyOffsets();
  }

  function syncStickyOffsets() {
    const header = table.querySelector("thead tr:first-child");
    const thead = table.querySelector("thead");
    if (!header) {
      return;
    }
    const project = header.querySelector(".col-project");
    const phase = header.querySelector(".col-phase");
    const task = header.querySelector(".col-task");
    const meta1 = header.querySelector(".col-meta1");
    const meta2 = header.querySelector(".col-meta2");
    const meta3 = header.querySelector(".col-meta3");
    if (!project || !phase || !meta1 || !meta2 || !meta3) {
      return;
    }
    const projectWidth = project.getBoundingClientRect().width;
    const phaseWidth = phase.getBoundingClientRect().width;
    const taskWidth = task ? task.getBoundingClientRect().width : 0;
    const meta1Width = meta1.getBoundingClientRect().width;
    const meta2Width = meta2.getBoundingClientRect().width;
    const meta3Width = meta3.getBoundingClientRect().width;
    const afterPhase = projectWidth + phaseWidth;
    const afterTask = afterPhase + taskWidth;
    const stickyColsWidth = afterTask + meta1Width + meta2Width + meta3Width;
    table.style.setProperty("--sticky-left-phase", `${projectWidth}px`);
    if (task) {
      table.style.setProperty("--sticky-left-task", `${afterPhase}px`);
    }
    table.style.setProperty("--sticky-left-meta1", `${afterTask}px`);
    table.style.setProperty("--sticky-left-meta2", `${afterTask + meta1Width}px`);
    table.style.setProperty("--sticky-left-meta3", `${afterTask + meta1Width + meta2Width}px`);
    table.style.setProperty("--sticky-cols-width", `${stickyColsWidth}px`);
    if (thead) {
      table.style.setProperty("--sticky-thead-height", `${thead.getBoundingClientRect().height}px`);
    }
  }

  function effortGrid() {
    if (actualMode) {
      return [];
    }
    return [...table.querySelectorAll("tbody tr")]
      .filter((row) => row.querySelector(".effort-input"))
      .map((row) => [...row.querySelectorAll(".effort-input")]);
  }

  function effortPosition(grid, input) {
    for (let row = 0; row < grid.length; row += 1) {
      const col = grid[row].indexOf(input);
      if (col >= 0) {
        return { row, col };
      }
    }
    return null;
  }

  function scrollEffortIntoView(input) {
    const scrollRoot = input.closest(".table-scroll");
    if (!scrollRoot) {
      return;
    }
    const stickyWidth = Number.parseFloat(getComputedStyle(table).getPropertyValue("--sticky-cols-width")) || 0;
    const stickyTop = Number.parseFloat(getComputedStyle(table).getPropertyValue("--sticky-thead-height")) || 0;
    const pad = 4;
    const rootRect = scrollRoot.getBoundingClientRect();
    const inputRect = input.getBoundingClientRect();
    const viewLeft = rootRect.left + scrollRoot.clientLeft + stickyWidth + pad;
    const viewRight = rootRect.left + scrollRoot.clientLeft + scrollRoot.clientWidth - pad;
    const viewTop = rootRect.top + scrollRoot.clientTop + stickyTop + pad;
    const viewBottom = rootRect.top + scrollRoot.clientTop + scrollRoot.clientHeight - pad;
    let deltaX = 0;
    let deltaY = 0;
    if (inputRect.left < viewLeft) {
      deltaX = inputRect.left - viewLeft;
    } else if (inputRect.right > viewRight) {
      deltaX = inputRect.right - viewRight;
    }
    if (inputRect.top < viewTop) {
      deltaY = inputRect.top - viewTop;
    } else if (inputRect.bottom > viewBottom) {
      deltaY = inputRect.bottom - viewBottom;
    }
    if (deltaX !== 0) {
      scrollRoot.scrollLeft += deltaX;
    }
    if (deltaY !== 0) {
      scrollRoot.scrollTop += deltaY;
    }
  }

  function focusEffortInput(input) {
    input.focus({ preventScroll: true });
    scrollEffortIntoView(input);
    input.select();
    requestAnimationFrame(() => {
      scrollEffortIntoView(input);
    });
  }

  function moveEffortInput(input, rowDelta, colDelta) {
    const grid = effortGrid();
    const pos = effortPosition(grid, input);
    if (!pos) {
      return;
    }
    const nextRow = pos.row + rowDelta;
    const nextCol = pos.col + colDelta;
    if (nextRow < 0 || nextRow >= grid.length) {
      return;
    }
    if (nextCol < 0 || nextCol >= grid[nextRow].length) {
      return;
    }
    focusEffortInput(grid[nextRow][nextCol]);
  }

  function handleEffortKeydown(event) {
    const key = event.key;
    if (key !== "ArrowLeft" && key !== "ArrowRight" && key !== "ArrowUp" && key !== "ArrowDown") {
      return;
    }
    event.preventDefault();
    if (key === "ArrowLeft") {
      moveEffortInput(event.target, 0, -1);
    } else if (key === "ArrowRight") {
      moveEffortInput(event.target, 0, 1);
    } else if (key === "ArrowUp") {
      moveEffortInput(event.target, -1, 0);
    } else if (key === "ArrowDown") {
      moveEffortInput(event.target, 1, 0);
    }
  }

  function scrollCurrentPeriodIntoView() {
    const head = table.querySelector("thead th.decade-head.col-current");
    if (!head) {
      return;
    }
    const stickyWidth = Number.parseFloat(getComputedStyle(table).getPropertyValue("--sticky-cols-width")) || 0;
    const pad = 8;
    const rootRect = root.getBoundingClientRect();
    const headRect = head.getBoundingClientRect();
    const viewLeft = rootRect.left + root.clientLeft + stickyWidth + pad;
    const viewRight = rootRect.left + root.clientLeft + root.clientWidth - pad;
    const viewWidth = Math.max(viewRight - viewLeft, headRect.width);
    const targetLeft = viewLeft + (viewWidth - headRect.width) / 2;
    root.scrollLeft += headRect.left - targetLeft;
  }

  function clearActiveColumn() {
    activeColEls.forEach((el) => el.classList.remove("col-editing"));
    activeColEls = [];
  }

  function setActiveColumn(input) {
    clearActiveColumn();
    const ym = input.dataset.ym;
    const decade = input.dataset.decade;
    if (!ym || !decade) {
      return;
    }
    const targets = [
      ...table.querySelectorAll(`.month-head[data-ym="${ym}"]`),
      ...table.querySelectorAll(`.decade-head[data-ym="${ym}"][data-decade="${decade}"]`),
    ];
    targets.forEach((el) => {
      el.classList.add("col-editing");
      activeColEls.push(el);
    });
  }

  function setActiveRow(input) {
    if (activeRow) {
      activeRow.classList.remove("row-editing");
    }
    activeRow = input.closest("tr");
    if (activeRow) {
      activeRow.classList.add("row-editing");
    }
    setActiveColumn(input);
  }

  function clearActiveRow() {
    if (activeRow) {
      activeRow.classList.remove("row-editing");
      activeRow = null;
    }
    clearActiveColumn();
  }

  function updatePeriodTotals() {
    if (table.classList.contains("schedule-detail")) {
      updateProcessRemainders();
      return;
    }
    const processTotals = {};
    table.querySelectorAll(".effort-input").forEach((input) => {
      const key = `${input.dataset.ym}-${input.dataset.decade}`;
      const value = Number(input.value || 0);
      processTotals[key] = (processTotals[key] || 0) + (Number.isNaN(value) ? 0 : value);
    });
    table.querySelectorAll("[data-period-allocated]").forEach((cell) => {
      const key = cell.getAttribute("data-period-allocated");
      const allocated = processTotals[key] || 0;
      cell.textContent = formatSum(allocated);
      const capacity = Number(cell.dataset.capacity || 0);
      cell.classList.remove("cap-warn", "cap-over");
      if (capacity <= 0) {
        if (allocated > 0) {
          cell.classList.add("cap-over");
        }
      } else if (allocated >= capacity) {
        cell.classList.add("cap-over");
      } else if (allocated >= capacity * 0.8) {
        cell.classList.add("cap-warn");
      }
    });
  }

  function updateProcessRemainders(processId) {
    const selector = processId
      ? `[data-process-remain^="${processId}|"]`
      : "[data-process-remain]";
    table.querySelectorAll(selector).forEach((cell) => {
      const key = cell.getAttribute("data-process-remain") || "";
      const parts = key.split("|");
      if (parts.length < 3) {
        return;
      }
      const pid = parts[0];
      const ym = parts[1];
      const decade = parts[2];
      let allocated = 0;
      table.querySelectorAll(`.effort-input[data-process-id="${pid}"][data-ym="${ym}"][data-decade="${decade}"]`).forEach((input) => {
        const value = Number(input.value || 0);
        allocated += Number.isNaN(value) ? 0 : value;
      });
      const budget = Number(cell.dataset.processBudget || 0);
      const remain = Math.round((budget - allocated) * 10) / 10;
      if (budget === 0 && allocated === 0) {
        cell.textContent = "";
        cell.classList.remove("cap-over");
        return;
      }
      cell.textContent = formatSum(remain);
      cell.classList.toggle("cap-over", remain < 0);
    });
  }

  function applyDiffClass(cell, diffSign) {
    if (!cell) {
      return;
    }
    cell.classList.remove("warn", "diff-over");
    if (diffSign > 0) {
      cell.classList.add("warn");
    } else if (diffSign < 0) {
      cell.classList.add("diff-over");
    }
  }

  function applyRowMeta(rowId, allocated, diff, diffSign) {
    const allocatedCell = table.querySelector(`[data-phase-allocated="${rowId}"]`);
    const diffCell = table.querySelector(`[data-phase-diff="${rowId}"]`);
    if (allocatedCell) {
      const single = allocatedCell.querySelector(".sum-single");
      if (single) {
        single.textContent = allocated;
      } else {
        allocatedCell.textContent = allocated;
      }
    }
    if (diffCell) {
      diffCell.textContent = diff;
      applyDiffClass(diffCell, diffSign);
    }
  }

  function cellIsActive(cell, hasEffort) {
    return Boolean(hasEffort) || cell.dataset.inRange === "1";
  }

  function saveAllocation(input) {
    if (actualMode) {
      return Promise.resolve();
    }
    const body = new URLSearchParams();
    body.set("csrf", csrf);
    body.set("task_id", input.dataset.taskId || "");
    body.set("project_process_id", input.dataset.projectProcessId || "");
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
      const effort = data.effort === "0.0" ? "" : data.effort;
      input.value = effort;
      input.dataset.estimate = effort;
      syncEffortStyle(input);
      const rowId = data.task_id || data.project_process_id || "";
      applyRowMeta(rowId, data.allocated, data.diff, data.diff_sign);
      updateProcessRemainders(input.dataset.processId || "");
      updatePeriodTotals();
    }).catch((err) => {
      window.alert(err.message || "保存に失敗しました");
    });
  }

  function queueSave(input) {
    if (actualMode) {
      return;
    }
    window.clearTimeout(saveTimer);
    saveTimer = window.setTimeout(() => {
      saveAllocation(input);
    }, 400);
  }

  inputs.forEach((input) => {
    input.addEventListener("keydown", (event) => {
      if (actualMode) {
        event.preventDefault();
        return;
      }
      handleEffortKeydown(event);
    });
    input.addEventListener("focus", () => {
      if (actualMode) {
        input.blur();
        return;
      }
      setActiveRow(input);
    });
    input.addEventListener("blur", () => clearActiveRow());
    input.addEventListener("change", () => {
      if (actualMode) {
        return;
      }
      window.clearTimeout(saveTimer);
      saveAllocation(input);
    });
    input.addEventListener("input", () => {
      if (actualMode) {
        return;
      }
      const cell = input.closest(".phase-cell");
      if (cell) {
        cell.classList.toggle("active", cellIsActive(cell, input.value && Number(input.value) > 0));
      }
      updateProcessRemainders(input.dataset.processId || "");
      queueSave(input);
    });
  });

  if (actualModeButton) {
    actualModeButton.addEventListener("click", () => {
      setActualMode(!actualMode);
    });
  }

  syncStickyOffsets();
  updateProcessRemainders();
  window.addEventListener("resize", syncStickyOffsets);
  requestAnimationFrame(() => {
    syncStickyOffsets();
    scrollCurrentPeriodIntoView();
  });
})();
