'use strict';

import "./shared.js"
import Choices from "choices.js"

import "../styles/create.css"

window.onload = function() {
    alert('heyyyy :)')
    document.querySelectorAll('select.choices')
        .forEach(el => {
            new Choices(el, {allowHTML: true})
        })
}
