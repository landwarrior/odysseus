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

  function closeNav(header, toggle) {
    if (!header || !toggle) {
      return;
    }
    header.classList.remove("nav-open");
    toggle.setAttribute("aria-expanded", "false");
    toggle.setAttribute("aria-label", "メニューを開く");
  }

  function setupHeaderMenus() {
    var header = document.querySelector("header.top");
    if (!header) {
      return;
    }
    var toggle = header.querySelector(".nav-toggle");
    var nav = header.querySelector(".site-nav");
    var userMenu = header.querySelector(".user-menu");

    if (toggle && nav) {
      toggle.addEventListener("click", function () {
        var open = header.classList.toggle("nav-open");
        toggle.setAttribute("aria-expanded", open ? "true" : "false");
        toggle.setAttribute("aria-label", open ? "メニューを閉じる" : "メニューを開く");
        if (open && userMenu) {
          userMenu.removeAttribute("open");
        }
      });
      nav.addEventListener("click", function (event) {
        if (event.target.closest("a")) {
          closeNav(header, toggle);
        }
      });
    }

    if (userMenu) {
      userMenu.addEventListener("toggle", function () {
        if (userMenu.open) {
          closeNav(header, toggle);
        }
      });
    }

    document.addEventListener("click", function (event) {
      if (!header.contains(event.target)) {
        closeNav(header, toggle);
        if (userMenu) {
          userMenu.removeAttribute("open");
        }
      }
    });

    document.addEventListener("keydown", function (event) {
      if (event.key !== "Escape") {
        return;
      }
      closeNav(header, toggle);
      if (userMenu) {
        userMenu.removeAttribute("open");
      }
    });
  }

  function currentTheme() {
    var theme = document.documentElement.getAttribute("data-theme");
    return theme === "dark" ? "dark" : "light";
  }

  function applyTheme(theme) {
    if (theme !== "dark" && theme !== "light") {
      theme = "light";
    }
    document.documentElement.setAttribute("data-theme", theme);
    try {
      localStorage.setItem("odysseus-theme", theme);
    } catch (e) {}
    document.querySelectorAll(".theme-option").forEach(function (button) {
      var on = button.getAttribute("data-theme-value") === theme;
      button.setAttribute("aria-pressed", on ? "true" : "false");
    });
  }

  function setupThemePicker() {
    applyTheme(currentTheme());
    document.querySelectorAll(".theme-option").forEach(function (button) {
      button.addEventListener("click", function () {
        applyTheme(button.getAttribute("data-theme-value"));
      });
    });
  }

  setupHeaderMenus();
  setupThemePicker();
})();
