major features:

- improve design
  - check mobile ui
  - add icons for social networks 
  - add time format 'X hours ago'
- edit comment
  - `PUT /api/v1/comment/{id}?site=site-id&url=post-url`
- add description of web part to readme

optimizations:

- rewrite fetcher if we need it (do we really need axios?)
- remove dev-tools if we have some
- remove babel-polyfill if we don't need it
  
  
minor features:
  
- add manual sort
  - `GET /api/v1/find?site=site-id&url=post-url&sort=fld&format=tree`
- add comments counter
  - `GET /api/v1/count?site=site-id&url=post-url`


other:

- check todos
