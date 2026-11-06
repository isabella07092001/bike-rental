// Dashboard (start page): fleet status right now and upcoming reservations.
// Read-only; the data is reloaded every time the dashboard is opened.
const dashboardPage = (() => {
  const errorLine = document.getElementById("dashboard-error");
  const generated = document.getElementById("dashboard-generated");
  const fleetBody = document.getElementById("dashboard-fleet");
  const upcomingBody = document.getElementById("dashboard-upcoming");

  const dateFmt = new Intl.DateTimeFormat("de-DE", { weekday: "short", day: "2-digit", month: "2-digit" });
  const timeFmt = new Intl.DateTimeFormat("de-DE", { hour: "2-digit", minute: "2-digit" });

  function formatPeriod(startISO, endISO) {
    const s = new Date(startISO);
    const e = new Date(endISO);
    const end = s.toDateString() === e.toDateString() ? timeFmt.format(e) : `${dateFmt.format(e)} ${timeFmt.format(e)}`;
    return `${dateFmt.format(s)} ${timeFmt.format(s)} – ${end}`;
  }

  function setText(id, value) {
    document.getElementById(id).textContent = value;
  }

  function cell(row, text, className = "") {
    const c = row.insertCell();
    c.textContent = text;
    if (className) c.className = className;
    return c;
  }

  function emptyRow(colSpan, text) {
    const row = document.createElement("tr");
    const c = row.insertCell();
    c.colSpan = colSpan;
    c.className = "empty";
    c.textContent = text;
    return row;
  }

  function render(s) {
    const f = s.fleet;
    setText("tile-total", f.total);
    setText("tile-available", f.availableNow);
    setText("tile-rented", f.rentedNow);
    setText("tile-maintenance", f.maintenance);
    generated.textContent = `As of ${timeFmt.format(new Date(s.generatedAt))}`;

    fleetBody.replaceChildren(
      ...f.byCategory.map((c) => {
        const row = document.createElement("tr");
        if (c.total === 0) row.className = "retired";
        cell(row, c.label);
        for (const n of [c.total, c.availableNow, c.rentedNow, c.maintenance]) cell(row, n, "num");
        return row;
      })
    );

    const u = s.upcoming;
    setText("dashboard-days", u.days);
    setText("dashboard-upcoming-count", `(${u.count})`);
    if (u.reservations.length === 0) {
      upcomingBody.replaceChildren(emptyRow(4, `No reservations in the next ${u.days} days.`));
      return;
    }
    upcomingBody.replaceChildren(
      ...u.reservations.map((r) => {
        const row = document.createElement("tr");
        cell(row, formatPeriod(r.start, r.end));
        cell(row, r.personName);
        cell(row, r.bikes.join(", "));
        cell(row, formatCents(r.totalPriceCents), "num");
        return row;
      })
    );
  }

  async function load() {
    errorLine.textContent = "";
    try {
      render(await api("GET", "/api/dashboard"));
    } catch (err) {
      errorLine.textContent = err.message;
    }
  }

  function init() {
    window.addEventListener("hashchange", () => {
      if (location.hash === "#dashboard" || location.hash === "") load();
    });
    load();
  }

  return { init };
})();

dashboardPage.init();
