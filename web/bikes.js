// Bikes page: list, create/edit form, retire.
const bikesPage = (() => {
  const statusLabels = { available: "Available", maintenance: "Maintenance", retired: "Retired" };
  let categories = []; // [{value, label, hourlyPriceCents}]

  const form = document.getElementById("bike-form");
  const fields = form.elements;
  const formTitle = document.getElementById("bike-form-title");
  const formError = document.getElementById("bike-form-error");
  const cancelBtn = document.getElementById("bike-cancel");
  const showRetired = document.getElementById("show-retired");
  const search = document.getElementById("bike-search");
  const categoryFilter = document.getElementById("bike-category-filter");
  const tableBody = document.getElementById("bike-rows");
  const listError = document.getElementById("bike-list-error");

  function categoryOf(value) {
    return categories.find((c) => c.value === value) || { label: value, hourlyPriceCents: 0 };
  }

  async function loadCategories() {
    categories = await api("GET", "/api/categories");
    fields.category.replaceChildren(
      ...categories.map((c) => new Option(`${c.label} (${formatCents(c.hourlyPriceCents)}/h)`, c.value))
    );
    categoryFilter.replaceChildren(new Option("All categories", ""), ...categories.map((c) => new Option(c.label, c.value)));
  }

  function isFiltered() {
    return search.value.trim() !== "" || categoryFilter.value !== "";
  }

  async function loadBikes() {
    listError.textContent = "";
    const params = new URLSearchParams();
    if (search.value.trim()) params.set("q", search.value.trim());
    if (categoryFilter.value) params.set("category", categoryFilter.value);
    try {
      const active = await api("GET", `/api/bikes?${params}`);
      params.set("status", "retired");
      const retired = showRetired.checked ? await api("GET", `/api/bikes?${params}`) : [];
      renderRows([...active, ...retired]);
    } catch (err) {
      listError.textContent = err.message;
    }
  }

  function renderRows(bikes) {
    if (bikes.length === 0) {
      const row = tableBody.insertRow();
      const cell = row.insertCell();
      cell.colSpan = 6;
      cell.className = "empty";
      cell.textContent = isFiltered() ? "No matching bikes." : "No bikes yet.";
      tableBody.replaceChildren(row);
      return;
    }
    tableBody.replaceChildren(...bikes.map(renderRow));
  }

  function renderRow(bike) {
    const cat = categoryOf(bike.category);
    const row = document.createElement("tr");
    if (bike.status === "retired") row.className = "retired";

    for (const text of [bike.name, cat.label, formatCents(cat.hourlyPriceCents)]) {
      row.insertCell().textContent = text;
    }
    const status = document.createElement("span");
    status.className = `badge ${bike.status}`;
    status.textContent = statusLabels[bike.status] || bike.status;
    row.insertCell().append(status);
    row.insertCell().textContent = bike.description;

    const actions = row.insertCell();
    actions.className = "actions";
    actions.append(button("Edit", () => startEdit(bike)));
    if (bike.status !== "retired") {
      actions.append(button("Retire", () => retire(bike), "danger"));
    }
    return row;
  }

  function button(label, onClick, className = "") {
    const b = document.createElement("button");
    b.type = "button";
    b.textContent = label;
    b.className = className;
    b.addEventListener("click", onClick);
    return b;
  }

  function startEdit(bike) {
    fields.id.value = bike.id;
    fields.name.value = bike.name;
    fields.category.value = bike.category;
    fields.status.value = bike.status;
    fields.description.value = bike.description;
    formTitle.textContent = `Edit ${bike.name}`;
    cancelBtn.hidden = false;
    formError.textContent = "";
    fields.name.focus();
  }

  function resetForm() {
    form.reset();
    fields.id.value = "";
    formTitle.textContent = "New bike";
    cancelBtn.hidden = true;
    formError.textContent = "";
  }

  async function submit(event) {
    event.preventDefault();
    const body = {
      name: fields.name.value,
      category: fields.category.value,
      status: fields.status.value,
      description: fields.description.value,
    };
    try {
      if (fields.id.value) {
        await api("PUT", `/api/bikes/${encodeURIComponent(fields.id.value)}`, body);
      } else {
        await api("POST", "/api/bikes", body);
      }
      resetForm();
      await loadBikes();
    } catch (err) {
      formError.textContent = err.message;
    }
  }

  async function retire(bike) {
    if (!confirm(`Retire "${bike.name}"? It will be hidden from the fleet but kept for history.`)) return;
    try {
      await api("POST", `/api/bikes/${encodeURIComponent(bike.id)}/retire`);
      if (fields.id.value === bike.id) resetForm();
      await loadBikes();
    } catch (err) {
      listError.textContent = err.message;
    }
  }

  async function init() {
    form.addEventListener("submit", submit);
    cancelBtn.addEventListener("click", resetForm);
    showRetired.addEventListener("change", loadBikes);
    categoryFilter.addEventListener("change", loadBikes);
    let timer;
    search.addEventListener("input", () => {
      clearTimeout(timer);
      timer = setTimeout(loadBikes, 250);
    });
    // Reservations may have changed what can be retired; reload when the page is opened.
    window.addEventListener("hashchange", () => {
      if (location.hash === "#bikes") loadBikes();
    });
    try {
      await loadCategories();
    } catch (err) {
      listError.textContent = err.message;
      return;
    }
    await loadBikes();
  }

  return { init };
})();

bikesPage.init();
