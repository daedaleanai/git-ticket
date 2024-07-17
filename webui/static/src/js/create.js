'use strict';

import "./shared.js"
import Choices from "choices.js"

import "../styles/create.css"

window.onload = function() {
    document.querySelectorAll('select.choices')
        .forEach(el => {
            new Choices(el, {allowHTML: true})
        })
}
