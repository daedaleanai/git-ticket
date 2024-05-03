"use strict";

window.onload = function () {
    const alertEl = document.getElementById("alert");
    alertEl.addEventListener("click", (e) => {
        alertEl.style.display = "none";
    });

    for (const ticketEl of document.getElementsByClassName("gt-ticket")) {
        ticketEl.draggable = true;
        ticketEl.addEventListener("dragstart", (e) => {
            e.dataTransfer.setData("text/plain", e.target.id);

            const placeholderEl = document.getElementById("drag-drop-placeholder");
            placeholderEl.style.display = "block";
            placeholderEl.style.height = e.target.getBoundingClientRect().height + 'px';
        });
    }

    for (const columnEl of document.getElementsByClassName("gt-column-body")) {
        columnEl.addEventListener("dragover", (e) => {
            e.preventDefault();
            e.dataTransfer.dropEffect = "move";

            const placeholderEl = document.getElementById("drag-drop-placeholder");
            for (const childEl of columnEl.children) {
                const rect = childEl.getBoundingClientRect()
                if ((rect.top + rect.bottom) / 2 > e.clientY) {
                    childEl.insertAdjacentElement('beforebegin', placeholderEl);
                    return;
                }
            }
            columnEl.insertAdjacentElement('beforeend', placeholderEl);
        });

        columnEl.addEventListener("drop", async (e) => {
            e.preventDefault();

            const ticket = e.dataTransfer.getData("text/plain");
            const status = columnEl.dataset.status;

            const resp = await fetch("/api", {
                method: "POST",
                body: JSON.stringify({
                    action: "setStatus",
                    ticket,
                    status,
                })
            });
            const text = await resp.text();
            
            const placeholderEl = document.getElementById("drag-drop-placeholder");
            placeholderEl.style.display = "none";

            const alertEl = document.getElementById("alert");
            alertEl.style.display = "block";
            alertEl.classList.remove("alert-danger");
            alertEl.classList.remove("alert-success");
            alertEl.innerText = text;

            if (resp.ok) {
                const ticketEl = document.getElementById(ticket);
                placeholderEl.insertAdjacentElement('beforebegin', ticketEl);
                alertEl.classList.add("alert-success");
            } else {
                alertEl.classList.add("alert-danger");
            }
        });
    }
};
