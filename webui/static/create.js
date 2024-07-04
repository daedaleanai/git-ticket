'use strict';

window.onload = function() {
    document.querySelectorAll('select.choices')
        .forEach(el => new Choices(el))
}
