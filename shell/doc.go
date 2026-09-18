/*
Package shell is the window skeleton: header, section navigation, one content
scroller, status bar, busy indicator, result banners, and the runner that keeps
core calls off the UI thread.

Ported from the shells of angou, nmsbonker and clockwork-orange (same author,
MIT). Where they disagreed, the popup form of banners and progress won, because
nothing transient may reflow the interface.

A program describes itself with Options (its sections, header actions, status
bar segments and load hooks) and calls Run. Sections are stateless build
functions: any change rebuilds the section from the program's state, so only
the builder needs to know every reason a button is disabled. A section that
holds live widgets or a scroll callback implements Detacher and is told before
its replacement is built.

The shell is a concrete type on purpose. Sections are the program's own code,
and Headless gives tests a shell with no window whose Perform runs inline.
*/
package shell
