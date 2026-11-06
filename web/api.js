// Small fetch wrapper for the Go JSON API. Throws an Error with a user-facing message:
// the server's {"error": "..."} text, or a clear note when the server is not reachable.
// A FormData body is sent as multipart/form-data (the browser sets the boundary);
// anything else as JSON.
async function api(method, path, body) {
  const options = { method, headers: {} };
  if (body instanceof FormData) {
    options.body = body;
  } else if (body !== undefined) {
    options.headers["Content-Type"] = "application/json";
    options.body = JSON.stringify(body);
  }
  let res;
  try {
    res = await fetch(path, options);
  } catch {
    throw new Error("The server cannot be reached. Is it running?");
  }
  const data = res.status === 204 ? null : await res.json().catch(() => null);
  if (!res.ok) {
    throw new Error((data && data.error) || `The request failed (HTTP ${res.status}). Please try again.`);
  }
  return data;
}

// Formats a price in cents as euros, e.g. 600 -> "6,00 €".
function formatCents(cents) {
  return (cents / 100).toLocaleString("de-DE", { style: "currency", currency: "EUR" });
}
