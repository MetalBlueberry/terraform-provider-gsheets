## Terraform provider google sheets

I was bored and I wanted to play around with implementing a provider. I had the idea to fetch the data from a google sheet where I already have a list of people that I want to give access to certain resources. Then use terraform to set the access.

It works, so I'm just saving the crappy code here. It is still far away from production ready. BUT.... it works on my machine.


## TODO

- Implement all the API fields
- Handle errors and rate limit errors with retries.