// Persons page: list with search, create (with ID copy), edit, replace ID copy, archive.
const personsPage = (() => {
  const form = document.getElementById("person-form");
  const fields = form.elements;
  const formTitle = document.getElementById("person-form-title");
  const formError = document.getElementById("person-form-error");
  const fileLabel = document.getElementById("person-file-label");
  const cancelBtn = document.getElementById("person-cancel");
  const search = document.getElementById("person-search");
  const showArchived = document.getElementById("show-archived");
  const tableBody = document.getElementById("person-rows");
  const listError = document.getElementById("person-list-error");
  const replaceInput = document.getElementById("person-replace-file");
  let replaceTarget = null; // person whose ID copy is being replaced

  async function loadPersons() {
    listError.textContent = "";
    const params = new URLSearchParams();
    if (search.value.trim()) params.set("q", search.value.trim());
    if (showArchived.checked) params.set("archived", "true");
    try {
      renderRows(await api("GET", `/api/persons?${params}`));
    } catch (err) {
      listError.textContent = err.message;
    }
  }

  function renderRows(persons) {
    if (persons.length === 0) {
      const row = document.createElement("tr");
      const cell = row.insertCell();
      cell.colSpan = 5;
      cell.className = "empty";
      cell.textContent = search.value.trim() ? "No matching persons." : showArchived.checked ? "No archived persons." : "No persons yet.";
      tableBody.replaceChildren(row);
      return;
    }
    tableBody.replaceChildren(...persons.map(renderRow));
  }

  function renderRow(p) {
    const row = document.createElement("tr");
    if (p.archived) row.className = "retired";

    row.insertCell().textContent = `${p.lastName}, ${p.firstName}`;
    row.insertCell().textContent = p.idNumber;
    row.insertCell().textContent = p.phone;

    const doc = row.insertCell();
    const link = document.createElement("a");
    link.href = `/api/persons/${encodeURIComponent(p.id)}/id-document`;
    link.target = "_blank";
    link.rel = "noopener";
    link.textContent = "View";
    link.title = `${p.idDocument.fileName} · ${formatSize(p.idDocument.size)} · uploaded ${new Date(p.idDocument.uploadedAt).toLocaleString("de-DE")}`;
    const kind = document.createElement("span");
    kind.className = "muted";
    kind.textContent = ` (${fileKind(p.idDocument.contentType)})`;
    doc.append(link, kind);

    const actions = row.insertCell();
    actions.className = "actions";
    actions.append(button("Edit", () => startEdit(p)));
    actions.append(button("Replace ID copy", () => startReplace(p)));
    if (p.archived) {
      actions.append(button("Reactivate", () => setArchived(p, false)));
    } else {
      actions.append(button("Archive", () => setArchived(p, true), "danger"));
    }
    return row;
  }

  function fileKind(contentType) {
    return { "application/pdf": "PDF", "image/jpeg": "JPG", "image/png": "PNG" }[contentType] || contentType;
  }

  function formatSize(bytes) {
    return bytes < 1024 * 1024 ? `${Math.ceil(bytes / 1024)} KB` : `${(bytes / 1024 / 1024).toFixed(1)} MB`;
  }

  function button(label, onClick, className = "") {
    const b = document.createElement("button");
    b.type = "button";
    b.textContent = label;
    b.className = className;
    b.addEventListener("click", onClick);
    return b;
  }

  function startEdit(p) {
    fields.id.value = p.id;
    fields.firstName.value = p.firstName;
    fields.lastName.value = p.lastName;
    fields.idNumber.value = p.idNumber;
    fields.phone.value = p.phone;
    // The ID copy is replaced separately; editing only changes the text fields.
    fileLabel.hidden = true;
    fields.idDocument.required = false;
    formTitle.textContent = `Edit ${p.firstName} ${p.lastName}`;
    cancelBtn.hidden = false;
    formError.textContent = "";
    fields.firstName.focus();
  }

  function resetForm() {
    form.reset();
    fields.id.value = "";
    fileLabel.hidden = false;
    fields.idDocument.required = true;
    formTitle.textContent = "New person";
    cancelBtn.hidden = true;
    formError.textContent = "";
  }

  async function submit(event) {
    event.preventDefault();
    formError.textContent = "";
    try {
      if (fields.id.value) {
        await api("PUT", `/api/persons/${encodeURIComponent(fields.id.value)}`, {
          firstName: fields.firstName.value,
          lastName: fields.lastName.value,
          idNumber: fields.idNumber.value,
          phone: fields.phone.value,
        });
      } else {
        // FormData(form) includes all named fields and the selected file (field "idDocument").
        const data = new FormData(form);
        data.delete("id");
        await api("POST", "/api/persons", data);
      }
      resetForm();
      await loadPersons();
    } catch (err) {
      formError.textContent = err.message;
    }
  }

  function startReplace(p) {
    replaceTarget = p;
    replaceInput.value = "";
    replaceInput.click();
  }

  async function replaceDocument() {
    const file = replaceInput.files[0];
    if (!replaceTarget || !file) return;
    const data = new FormData();
    data.append("idDocument", file);
    try {
      await api("PUT", `/api/persons/${encodeURIComponent(replaceTarget.id)}/id-document`, data);
      await loadPersons();
    } catch (err) {
      listError.textContent = `${replaceTarget.firstName} ${replaceTarget.lastName}: ${err.message}`;
    } finally {
      replaceTarget = null;
    }
  }

  async function setArchived(p, archived) {
    const name = `${p.firstName} ${p.lastName}`;
    if (archived && !confirm(`Archive ${name}? The person will be hidden but kept, including the ID copy.`)) return;
    try {
      await api("POST", `/api/persons/${encodeURIComponent(p.id)}/${archived ? "archive" : "reactivate"}`);
      if (fields.id.value === p.id) resetForm();
      await loadPersons();
    } catch (err) {
      listError.textContent = err.message;
    }
  }

  function init() {
    form.addEventListener("submit", submit);
    cancelBtn.addEventListener("click", resetForm);
    showArchived.addEventListener("change", loadPersons);
    window.addEventListener("hashchange", () => {
      if (location.hash === "#persons") loadPersons();
    });
    replaceInput.addEventListener("change", replaceDocument);
    let timer;
    search.addEventListener("input", () => {
      clearTimeout(timer);
      timer = setTimeout(loadPersons, 250);
    });
    loadPersons();
  }

  return { init };
})();

personsPage.init();
