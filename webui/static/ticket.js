'use strict';

async function submitComment() {
    const ticket = document.getElementById('ticketId').textContent;
    const comment = document.getElementById('commentFormControlTextArea').value;
    const resp = await fetch('/api/submit-comment', {
        method: 'POST',
        body: JSON.stringify({
            'ticket': ticket,
            'comment': comment,
        })
    });
    const responseText = await resp.text();


    if (resp.ok) {
        location.reload()
    } else {
        const alertEl = document.getElementById('alert');
        alertEl.style.display = 'block';
        alertEl.innerText = responseText;
    }
};

window.onload = function() {
    const alertEl = document.getElementById('alert');
    alertEl.addEventListener('click', (e) => {
        alertEl.style.display = 'none';
    });
};
