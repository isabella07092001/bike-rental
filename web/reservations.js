// Reservations page: list, create/edit with available bikes and live price preview, cancel.
const reservationsPage = (() => {
  const phaseLabels = { upcoming: "Upcoming", active: "Active", completed: "Completed", cancelled: "Cancelled" };

  const form = document.getElementById("reservation-form");
  const fields = form.elements;
  const formTitle = document.getElementById("reservation-form-title");
  const formError = document.getElementById("reservation-form-error");
  const submitBtn = form.querySelector("button[type=submit]");
  const cancelEditBtn = document.getElementById("reservation-cancel-edit");
  const bikeBox = document.getElementById("reservation-bikes");
  const quoteLine = document.getElementById("reservation-quote");
  const phaseFilter = document.getElementById("reservation-phase");
  const search = document.getElementById("reservation-search");
  const tableBody = document.getElementById("reservation-rows");
  const listError = document.getElementById("reservation-list-error");

  let editing = null; // reservation being edited, or null
  let refreshTimer;

  // ---- time helpers (inputs use local time, the API RFC 3339) ----

  function pad(n) {
    return String(n).padStart(2, "0");
  }

  function toLocalInput(date) {
    return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
  }

  function inputToISO(value) {
    const d = new Date(value);
    return Number.isNaN(d.getTime()) ? "" : d.toISOString();
  }

  const dateFmt = new Intl.DateTimeFormat("de-DE", { weekday: "short", day: "2-digit", month: "2-digit", year: "numeric" });
  const timeFmt = new Intl.DateTimeFormat("de-DE", { hour: "2-digit", minute: "2-digit" });

  function formatDateTime(d) {
    return `${dateFmt.format(d)} ${timeFmt.format(d)}`;
  }

  function formatPeriod(startISO, endISO) {
    const s = new Date(startISO);
    const e = new Date(endISO);
    const sameDay = s.toDateString() === e.toDateString();
    return sameDay ? `${formatDateTime(s)} – ${timeFmt.format(e)}` : `${formatDateTime(s)} – ${formatDateTime(e)}`;
  }

  function period() {
    return { start: inputToISO(fields.start.value), end: inputToISO(fields.end.value) };
  }

  function selectedBikeIds() {
    return [...bikeBox.querySelectorAll("input[type=checkbox]:checked")].map((c) => c.value);
  }

  // ---- form: persons, available bikes, price preview ----

  async function loadPersons() {
    const persons = await api("GET", "/api/persons");
    const current = fields.personId.value;
    const options = persons.map((p) => new Option(`${p.lastName}, ${p.firstName} (${p.idNumber})`, p.id));
    fields.personId.replaceChildren(new Option(persons.length ? "Select a person…" : "No active persons – add one first", ""), ...options);
    fields.personId.value = current;
  }

  function scheduleRefresh() {
    clearTimeout(refreshTimer);
    refreshTimer = setTimeout(refreshBikesAndQuote, 200);
  }

  async function refreshBikesAndQuote() {
    await refreshBikes();
    await refreshQuote();
  }

  async function refreshBikes() {
    const { start, end } = period();
    const keep = new Set(selectedBikeIds());
    const legend = bikeBox.querySelector("legend");
    if (!start || !end) {
      bikeBox.replaceChildren(legend, note("Choose start and end to see the available bikes."));
      return;
    }
    const params = new URLSearchParams({ start, end });
    if (editing) params.set("excludeReservation", editing.id);
    let bikes;
    try {
      bikes = await api("GET", `/api/bikes/available?${params}`);
    } catch (err) {
      bikeBox.replaceChildren(legend, note(err.message, "error"));
      return;
    }
    if (bikes.length === 0) {
      bikeBox.replaceChildren(legend, note("No bike is available in this period."));
      return;
    }
    const categories = await getCategories();
    bikeBox.replaceChildren(
      legend,
      ...bikes.map((b) => {
        const cat = categories.find((c) => c.value === b.category);
        const label = document.createElement("label");
        const box = document.createElement("input");
        box.type = "checkbox";
        box.value = b.id;
        box.checked = keep.has(b.id);
        box.addEventListener("change", refreshQuote);
        const text = document.createElement("span");
        text.textContent = `${b.name} · ${cat ? cat.label : b.category} · ${formatCents(cat ? cat.hourlyPriceCents : 0)}/h`;
        label.append(box, text);
        return label;
      })
    );
  }

  let categoriesCache;
  async function getCategories() {
    if (!categoriesCache) categoriesCache = await api("GET", "/api/categories");
    return categoriesCache;
  }

  function note(text, className = "muted") {
    const p = document.createElement("p");
    p.className = className;
    p.textContent = text;
    return p;
  }

  async function refreshQuote() {
    const { start, end } = period();
    const bikeIds = selectedBikeIds();
    quoteLine.className = "quote";
    if (!start || !end || bikeIds.length === 0) {
      quoteLine.textContent = "";
      return;
    }
    try {
      const body = { bikeIds, start, end };
      if (editing) body.reservationId = editing.id;
      const q = await api("POST", "/api/reservations/quote", body);
      const hours = `${q.billedHours} started hour${q.billedHours === 1 ? "" : "s"}`;
      quoteLine.textContent = `${hours} → ${formatCents(q.totalPriceCents)}`;
      if (q.conflicts.length) {
        quoteLine.textContent += ` · Not bookable: ${q.conflicts.join("; ")}`;
        quoteLine.classList.add("error");
      }
    } catch (err) {
      quoteLine.textContent = err.message;
      quoteLine.classList.add("error");
    }
  }

  function setDefaultPeriod() {
    const start = new Date();
    start.setDate(start.getDate() + 1);
    start.setHours(10, 0, 0, 0);
    const end = new Date(start);
    end.setHours(12);
    fields.start.value = toLocalInput(start);
    fields.end.value = toLocalInput(end);
  }

  function resetForm() {
    editing = null;
    form.reset();
    fields.personId.disabled = false;
    formTitle.textContent = "New reservation";
    submitBtn.textContent = "Book";
    cancelEditBtn.hidden = true;
    formError.textContent = "";
    bikeBox.querySelectorAll("input[type=checkbox]").forEach((c) => (c.checked = false));
    setDefaultPeriod();
    refreshBikesAndQuote();
  }

  async function startEdit(r) {
    editing = r;
    formTitle.textContent = `Edit reservation of ${r.personName}`;
    submitBtn.textContent = "Save changes";
    cancelEditBtn.hidden = false;
    formError.textContent = "";
    fields.personId.value = r.personId;
    if (fields.personId.value !== r.personId) {
      fields.personId.append(new Option(r.personName, r.personId));
      fields.personId.value = r.personId;
    }
    fields.personId.disabled = true; // the person of a reservation cannot change
    fields.start.value = toLocalInput(new Date(r.start));
    fields.end.value = toLocalInput(new Date(r.end));
    bikeBox.replaceChildren(bikeBox.querySelector("legend"));
    // Pre-select the reservation's bikes after loading the available ones.
    await refreshBikes();
    for (const b of r.bikes) {
      const box = bikeBox.querySelector(`input[value="${CSS.escape(b.bikeId)}"]`);
      if (box) box.checked = true;
    }
    await refreshQuote();
    form.scrollIntoView({ behavior: "smooth" });
  }

  async function submit(event) {
    event.preventDefault();
    formError.textContent = "";
    const { start, end } = period();
    const body = { bikeIds: selectedBikeIds(), start, end };
    try {
      if (editing) {
        await api("PUT", `/api/reservations/${encodeURIComponent(editing.id)}`, body);
      } else {
        body.personId = fields.personId.value;
        await api("POST", "/api/reservations", body);
      }
      resetForm();
      await loadReservations();
    } catch (err) {
      formError.textContent = err.message;
    }
  }

  // ---- list ----

  async function loadReservations() {
    listError.textContent = "";
    const params = new URLSearchParams();
    if (phaseFilter.value) params.set("phase", phaseFilter.value);
    if (search.value.trim()) params.set("q", search.value.trim());
    try {
      renderRows(await api("GET", `/api/reservations?${params}`));
    } catch (err) {
      listError.textContent = err.message;
    }
  }

  function renderRows(reservations) {
    if (reservations.length === 0) {
      const row = document.createElement("tr");
      const cell = row.insertCell();
      cell.colSpan = 6;
      cell.className = "empty";
      cell.textContent = search.value.trim() ? "No matching reservations." : "No reservations.";
      tableBody.replaceChildren(row);
      return;
    }
    tableBody.replaceChildren(...reservations.map(renderRow));
  }

  function renderRow(r) {
    const row = document.createElement("tr");
    if (r.phase === "cancelled" || r.phase === "completed") row.className = "retired";

    row.insertCell().textContent = formatPeriod(r.start, r.end);

    const person = row.insertCell();
    person.textContent = r.personName;
    person.append(document.createElement("br"), muted(`ID ${r.idNumber}`));

    row.insertCell().textContent = r.bikes.map((b) => b.name).join(", ");

    const price = row.insertCell();
    price.className = "nowrap";
    price.textContent = formatCents(r.totalPriceCents);
    price.append(document.createElement("br"), muted(`${r.billedHours} h`));

    const badge = document.createElement("span");
    badge.className = `badge phase-${r.phase}`;
    badge.textContent = phaseLabels[r.phase] || r.phase;
    row.insertCell().append(badge);

    const actions = row.insertCell();
    actions.className = "actions";
    if (r.status === "booked" && r.phase === "upcoming") {
      const hint = r.canChange
        ? `Changes possible until ${formatDateTime(new Date(r.changeDeadline))}`
        : `Changes were only possible until ${formatDateTime(new Date(r.changeDeadline))} (24 h before start)`;
      actions.append(button("Edit", () => startEdit(r), "", !r.canChange, hint));
      actions.append(button("Cancel", () => cancel(r), "danger", !r.canChange, hint));
    }
    return row;
  }

  function muted(text) {
    const s = document.createElement("span");
    s.className = "muted";
    s.textContent = text;
    return s;
  }

  function button(label, onClick, className, disabled, title) {
    const b = document.createElement("button");
    b.type = "button";
    b.textContent = label;
    b.className = className;
    b.disabled = disabled;
    b.title = title;
    b.addEventListener("click", onClick);
    return b;
  }

  async function cancel(r) {
    if (!confirm(`Cancel the reservation of ${r.personName} (${formatPeriod(r.start, r.end)})?`)) return;
    try {
      await api("POST", `/api/reservations/${encodeURIComponent(r.id)}/cancel`);
      if (editing && editing.id === r.id) resetForm();
      await loadReservations();
      await refreshBikesAndQuote();
    } catch (err) {
      listError.textContent = err.message;
    }
  }

  // ---- init ----

  async function show() {
    try {
      await loadPersons();
    } catch (err) {
      formError.textContent = err.message;
    }
    await Promise.all([loadReservations(), refreshBikesAndQuote()]);
  }

  function init() {
    form.addEventListener("submit", submit);
    cancelEditBtn.addEventListener("click", resetForm);
    phaseFilter.addEventListener("change", loadReservations);
    let searchTimer;
    search.addEventListener("input", () => {
      clearTimeout(searchTimer);
      searchTimer = setTimeout(loadReservations, 250);
    });
    fields.start.addEventListener("input", scheduleRefresh);
    fields.end.addEventListener("input", scheduleRefresh);
    // Persons and bikes may have changed on the other pages.
    window.addEventListener("hashchange", () => {
      if (location.hash === "#reservations") show();
    });
    setDefaultPeriod();
    show();
  }

  return { init };
})();

reservationsPage.init();
