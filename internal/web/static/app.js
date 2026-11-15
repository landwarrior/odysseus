(function () {
  function stripComma(value) {
    return String(value).replace(/[,，]/g, "");
  }

  function withComma(value) {
    var raw = stripComma(value).replace(/[¥￥円\s\u3000]/g, "");
    if (raw === "") {
      return "";
    }
    if (!/^\d+$/.test(raw)) {
      return stripComma(value);
    }
    var out = "";
    for (var i = 0; i < raw.length; i++) {
      if (i > 0 && (raw.length - i) % 3 === 0) {
        out += ",";
      }
      out += raw.charAt(i);
    }
    return out;
  }

  function isYenInput(el) {
    return el && el.matches && el.matches("input.yen-input");
  }

  document.addEventListener("focusin", function (event) {
    if (!isYenInput(event.target)) {
      return;
    }
    event.target.value = stripComma(event.target.value);
  });

  document.addEventListener("focusout", function (event) {
    if (!isYenInput(event.target)) {
      return;
    }
    event.target.value = withComma(event.target.value);
  });
})();
