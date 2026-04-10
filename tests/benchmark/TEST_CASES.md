# PinchTab Benchmark Test Cases

39 test cases covering realistic browser automation scenarios against local fixture pages.

| # | Task | Description |
|---|------|-------------|
| **Group 0: Setup** | | |
| 0.1 | Health check | Confirm PinchTab is running with active instance |
| 0.2 | Fixtures reachable | Navigate to fixtures root, confirm accessible |
| **Group 1: Reading & Extracting** | | |
| 1.1 | Wiki categories | Extract category names + article counts from wiki index |
| 1.2 | Click a link | From wiki, click through to Go article |
| 1.3 | Table extraction | Read infobox — who designed Go, what year |
| 1.4 | Count list items | Count key features on Go article (expect 6) |
| 1.5 | Article headlines | Navigate to articles page, list all titles |
| 1.6 | Dashboard metrics | Extract Total Users, Revenue, Conversion Rate |
| **Group 2: Search & Dynamic** | | |
| 2.1 | Wiki search | Use search form to find "golang" |
| 2.2 | No results search | Search for nonexistent term, verify graceful |
| 2.3 | AI content search | Search for "artificial intelligence" |
| **Group 3: Form** | | |
| 3.1 | Complete form | Fill all fields + submit (name, email, phone, country, subject, message, checkbox, radio) |
| 3.2 | Reset/refill | After submit, verify reset button exists |
| **Group 4: SPA** | | |
| 4.1 | Read app state | Count tasks: 3 total, 2 active, 1 done |
| 4.2 | Add task | Add "Automate deployment" with high priority |
| 4.3 | Delete task | Delete first task, verify count changed |
| **Group 5: Login** | | |
| 5.1 | Invalid login | Try wrong credentials, verify error |
| 5.2 | Valid login | Login with correct creds, verify dashboard |
| **Group 6: E-commerce** | | |
| 6.1 | Research products | List products + prices, identify out-of-stock |
| 6.2 | Add to cart | Add 2 items, verify $449.98 total |
| 6.3 | Checkout | Complete order, verify confirmation |
| **Group 7: Content + Interaction** | | |
| 7.1 | Read & comment | Read Go article, post comment with 5-star rating |
| 7.2 | Cross-page research | Find biggest category on wiki, navigate to an article in it |
| **Group 8: Error Handling** | | |
| 8.1 | 404 handling | Navigate to missing page, verify no crash |
| 8.2 | Missing element | Click nonexistent selector, verify clear error |
| **Group 9: Export** | | |
| 9.1 | Screenshot | Take screenshot of dashboard |
| 9.2 | PDF export | Export dashboard as PDF |
| **Group 10: Modals** | | |
| 10.1 | Open modal | Click Settings on dashboard, verify modal |
| 10.2 | Modal interaction | Change theme to Dark, save, verify applied |
| **Group 11: Persistence** | | |
| 11.1 | State after reload | Add task, reload SPA, verify task persists |
| 11.2 | Logout/re-login | Sign out, log back in, verify session renewed |
| **Group 12: Multi-page Nav** | | |
| 12.1 | Navigate & return | Home -> wiki -> Go -> back -> back -> home |
| 12.2 | Cross-page compare | Compare article counts between wiki and articles page |
| **Group 13: Form Validation** | | |
| 13.1 | Required field | Submit without email, verify blocked |
| 13.2 | Optional field | Submit without phone, verify success |
| **Group 14: Dynamic Content** | | |
| 14.1 | Load more | Click "Load More Products" on ecommerce |
| 14.2 | Lazy-loaded item | Add a lazy-loaded product to cart |
| **Group 15: Data Aggregation** | | |
| 15.1 | Financial calc | Extract revenue + profit, calculate margin |
| 15.2 | Multi-page comparison | Compare features across 3 wiki language pages |
