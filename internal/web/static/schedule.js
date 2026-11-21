(() => {
  const root = document.querySelector(".table-scroll");
  const table = document.querySelector(".schedule-table");
  if (!root || !table) {
    return;
  }
  const csrf = root.getAttribute("data-csrf") || "";
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
    const decadeHeads = [...table.querySelectorAll("thead th.decade-head")];
    table.querySelectorAll("tr.project-summary").forEach((row) => {
      const cells = [...row.querySelectorAll("[data-project-allocated]")];
      cells.forEach((cell, index) => {
        const head = decadeHeads[index];
        if (!head) {
          return;
        }
        const key = `${head.dataset.ym}-${head.dataset.decade}`;
        cell.textContent = formatSum(totals[key] || 0);
      });
    });
    updateCapacityWarnings();
  }

  function updateCapacityWarnings(processId) {
    const selector = processId
      ? `[data-process-cap^="${processId}|"]`
      : "[data-process-cap]";
    table.querySelectorAll(selector).forEach((cell) => {
      const key = cell.getAttribute("data-process-cap") || "";
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

  function applyTaskMeta(taskId, allocated, diff, diffSign) {
    const allocatedCell = table.querySelector(`[data-phase-allocated="${taskId}"]`);
    const diffCell = table.querySelector(`[data-phase-diff="${taskId}"]`);
    if (allocatedCell) {
      allocatedCell.textContent = allocated;
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
      input.dataset.estimate = input.value;
      const cell = input.closest(".phase-cell");
      if (cell) {
        cell.classList.toggle("active", cellIsActive(cell, input.value));
      }
      applyTaskMeta(data.task_id, data.allocated, data.diff, data.diff_sign);
      updateCapacityWarnings(input.dataset.processId || "");
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
    input.addEventListener("keydown", handleEffortKeydown);
    input.addEventListener("focus", () => setActiveRow(input));
    input.addEventListener("blur", () => clearActiveRow());
    input.addEventListener("change", () => {
      window.clearTimeout(saveTimer);
      saveAllocation(input);
    });
    input.addEventListener("input", () => {
      const cell = input.closest(".phase-cell");
      if (cell) {
        cell.classList.toggle("active", cellIsActive(cell, input.value && Number(input.value) > 0));
      }
      updateCapacityWarnings(input.dataset.processId || "");
      queueSave(input);
    });
  });

  syncStickyOffsets();
  updateCapacityWarnings();
  window.addEventListener("resize", syncStickyOffsets);
  requestAnimationFrame(() => {
    syncStickyOffsets();
    scrollCurrentPeriodIntoView();
  });
})();
