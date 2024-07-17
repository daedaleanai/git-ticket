'use strict';

import "./shared.js"
import "../styles/checklist.css"

window.onload = function() {
    const selectReviewer = tabIdx => {
        for (const tabEl of document.getElementsByClassName('nav-link')) {
            import "../styles/checklist.css"
            tabEl.classList.remove('active');
        }

        const tabEl = document.getElementById('tab-' + tabIdx)
        tabEl.classList.add('active');

        for (const checklistEl of document.getElementsByClassName('checklist')) {
            checklistEl.style.display = 'none';
        }
        const checklistEl = document.getElementById('checklist-' + tabIdx)
        checklistEl.style.display = 'block';
    }
    
    for (const el of document.getElementsByClassName('checklist-state')) {
        el.addEventListener('mouseenter', (e) => {
            const usr = el.getAttribute('data-usr-id');
            const section = el.getAttribute('data-section-id');
            const question = el.getAttribute('data-question-id');
            const id =
                'comment-' + usr + '-sec-' + section + '-question-' + question;

            const commentEl = document.getElementById(
                'comment-' + usr + '-sec-' + section + '-question-' + question);

            if (commentEl.innerText !== '') {
                const alertEl = document.getElementById('alert');
                alertEl.style.display = 'block';
                alertEl.innerText = commentEl.innerText;
            }
        });

        el.addEventListener('mouseleave', (e) => {
            const alertEl = document.getElementById('alert');
            alertEl.style.display = 'none';
        });
    }
};
