// Minimal client-side navigation: shows the section matching the URL hash.
const pages = ["dashboard", "bikes", "persons", "reservations"];

function showPage() {
  const current = pages.includes(location.hash.slice(1)) ? location.hash.slice(1) : "dashboard";
  for (const id of pages) {
    document.getElementById(id).hidden = id !== current;
  }
  for (const link of document.querySelectorAll("nav a")) {
    link.classList.toggle("active", link.getAttribute("href") === "#" + current);
  }
}

async function checkHealth() {
  const el = document.getElementById("api-status");
  try {
    const res = await fetch("/api/health");
    const body = await res.json();
    el.textContent = "API: " + body.status;
    el.classList.toggle("ok", res.ok);
  } catch (err) {
    el.textContent = "API: unreachable";
    el.classList.add("error");
  }
}

window.addEventListener("hashchange", showPage);
showPage();
checkHealth();
