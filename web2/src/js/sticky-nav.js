;(function () {
  /*
    When the viewer scrolls the browser window, check to see if we
    need to "stick" the navigation header to the top of the viewport,
    or re-attach it back to its relative position in the document flow.
  */
  document.addEventListener('scroll', function(event) {
    var nav = document.getElementById("sticky-nav");
    var hud = document.getElementById("hud");
    if (!hud) { return; }

    var y = hud.clientHeight + hud.offsetTop;

    if (y - 29 <= window.scrollY) {
      nav.className = 'nav sticky on';
    } else if (y - 29 > window.scrollY) {
      nav.className = 'nav sticky';
    }
  });
})()
