(function () {
  const form = document.querySelector(".time-entry-form");
  const totalEls = document.querySelectorAll("[data-time-total]");
  if (!form || totalEls.length === 0) {
    return;
  }

  function parseHours(raw) {
    const s = String(raw ?? "").trim();
    if (s === "") {
      return 0;
    }
    const n = Number(s);
    if (!Number.isFinite(n) || n < 0) {
      return 0;
    }
    return Math.round(n * 10) / 10;
  }

  function formatHours(n) {
    const tenths = Math.round(n * 10);
    if (tenths % 10 === 0) {
      return String(tenths / 10);
    }
    return (tenths / 10).toFixed(1);
  }

  function refreshTotal() {
    // 代行時など、フォームに出ない同日実績は data-time-base に載せる。
    let sum = parseHours(form.getAttribute("data-time-base"));
    form.querySelectorAll(".time-hours input").forEach((input) => {
      if (input.disabled) {
        return;
      }
      sum += parseHours(input.value);
    });
    const text = formatHours(sum);
    totalEls.forEach((el) => {
      el.textContent = text;
    });
  }

  form.addEventListener("input", refreshTotal);
  form.addEventListener("change", refreshTotal);
  refreshTotal();
})();
