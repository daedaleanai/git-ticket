'use strict';

async function submitComment() {
    const ticket = document.getElementById('ticket-id').textContent.trim();
    const comment =
        document.getElementById('comment-form-control-text-area').value.trim();

    if (comment.length === 0) {
        const alertEl = document.getElementById('alert');
        alertEl.style.display = 'block';
        alertEl.innerText = 'did not submit empty comment';
        return
    }

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
