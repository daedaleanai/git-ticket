window.onload = function () {
    let settings = JSON.parse(localStorage.getItem('git-ticket'));
    if (settings == null) {
        localStorage.setItem('git-ticket', '{"bookmarks": {"all tickets": {"all tickets": ""}}}');
        settings = JSON.parse(localStorage.getItem('git-ticket'));
    }

    const bookmarks = settings.bookmarks;

    const menuEl = document.getElementById("menu");

    const urlParams = new URLSearchParams(window.location.search);
    const query = urlParams.get('q');

    for (const group in bookmarks) {
        let groupEl = document.createElement('div');
        groupEl.classList.add("gt-menu-group");
        menuEl.prepend(groupEl);

        let spanEl = document.createElement('span');
        spanEl.innerText = group;
        groupEl.appendChild(spanEl);

        for (const bookmark in bookmarks[group]) {
            const link = bookmarks[group][bookmark];

            let linkEl = document.createElement('a');
            linkEl.href = "?q=" + link;
            groupEl.appendChild(linkEl);

            let bookmarkEl = document.createElement('div');
            bookmarkEl.classList.add("gt-menu-bookmark");
            if (link == query) {
                bookmarkEl.classList.add("active");
            }
            bookmarkEl.innerText = bookmark;
            linkEl.appendChild(bookmarkEl);
        }
    }
};
