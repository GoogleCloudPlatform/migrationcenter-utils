# Example Terraform deployment for MC2BQ

## Configuration

For the full list of configuration values look at [variables.tf](./variables.tf).

The only mandatory variable is `project` which is the project where the Migration Center data is and where to set up the appropriate cloud resources.
`region` is an optional variable, which its default value is us-central1.

## Deploy

To deploy simply execute:

```sh
terraform apply
```

To deploy to a non-default region, execute:

```sh
terraform apply -var="region=some-region"
```
