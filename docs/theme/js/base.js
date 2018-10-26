function getSearchTerm() {
  const sPageURL = window.location.search.substring(1);
  const sURLVariables = sPageURL.split('&');

  for (let i = 0; i < sURLVariables.length; i++) {
    let sParameterName = sURLVariables[i].split('=');

    if (sParameterName[0] == 'q') {
      return sParameterName[1];
    }
  }
}

$(document).ready(function() {


  // SIDE NAV HIGHLIGHTING
  const $sideNavSecondaryMenu = $(".side-nav-secondary-buttons");
  const $sideNavSecondaryButtons = $sideNavSecondaryMenu.find('a');

  $sideNavSecondaryMenu.on("click", "a", (e) => {
    const $button = $(e.currentTarget);
    const isActive = $button.hasClass('is-active');

    setTimeout(() => {
      if(!isActive) {
        $sideNavSecondaryButtons.removeClass('is-active');
        $button.addClass("is-active");
      }
    }, 50);
  });

  // ACTIVATE CODE FORMATTING
  if(window.PR && window.PR.prettyPrint) {
    let $preTags = $('pre');

    $preTags.each((index, el) => {
      const $pre = $(el);
      const $code = $pre.find('code');
      const lang = $code.attr('class');

      $pre.addClass(`prettyprint lang-${lang}`);
    });

    window.PR.prettyPrint();
  }

  /* Prevent disabled links from causing a page reload */
  $("li.disabled a").click((e) => {
    e.preventDefault();
  });

    // ACTIVATE NAVIGATION
  if (window.grvlib) {
    new grvlib.TopNav();
    new grvlib.SecondaryNav();
    new grvlib.SideNav();
    grvlib.buttonSmoothScroll(16);
    grvlib.buttonRipple();
  }
});

