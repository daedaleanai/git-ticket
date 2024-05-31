'use strict';

function selectReviewer(tabIdx) {
    for (const tabEl of document.getElementsByClassName('nav-link')) {
        tabEl.classList.remove('active');
    }

    const tabEl = document.getElementById('tab-' + tabIdx)
    tabEl.classList.add('active');

    for (const checklistEl of document.getElementsByClassName('checklist')) {
        checklistEl.style.display = 'none';
    }
    const checklistEl = document.getElementById('checklist-' + tabIdx)
    checklistEl.style.display = 'block';
};
