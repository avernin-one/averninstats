let darkButton = "🌑"
let lightButton = "🌕"
let lastModeKey = "lastMode"

let toggleButton = document.querySelector("#toggleButton");
toggleButton.addEventListener("click", (event) => {
    document.body.classList.toggle("light");

    if (document.body.classList.contains("light")) {
        toggleButton.innerText = darkButton
        localStorage.setItem(lastModeKey, "light")
    } else {
        toggleButton.innerText = lightButton
        localStorage.setItem(lastModeKey, "dark")
    }
});

window.addEventListener("DOMContentLoaded", (event) => {
    lastMode = localStorage.getItem(lastModeKey)

    if (lastMode && lastMode == "light" || window.matchMedia && window.matchMedia('(prefers-color-scheme: light)').matches) {
        document.body.classList.add("light");
        toggleButton.innerText = darkButton
        localStorage.setItem(lastModeKey, "light")
    } else {
        toggleButton.innerText = lightButton
        localStorage.setItem(lastModeKey, "dark")
    }

});


let searchInput = document.querySelector("input[name='search_input']")
searchInput.addEventListener("input", (event) => {
    search = event.target.value.toLowerCase();
    sectionMenu = document.querySelector("aside[class='section_menu']")
    elements = sectionMenu.querySelectorAll("div.entry");
    elements.forEach((e) => {
        let anchor = e.querySelector("a");
        let text = anchor.innerText.toLowerCase();

        if (search == "" || text.includes(search)) {
            e.classList.remove("hide");
        } else {
            e.classList.add("hide");
        }
    });
});

let menuLinks = document.querySelectorAll(".entry a");
menuLinks.forEach(function(el) {
    el.addEventListener("click", (event) => {
        let target = document.getElementById(event.target.hash.replace("#", ""));

        if (target && target != null) {
            target.classList.remove("glow");
            target.classList.add("glow");
        }
    });
});
