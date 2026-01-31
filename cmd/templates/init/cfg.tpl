# aplcli config
defaults:
  - &email {{ .email }}
  - &region {{ .region }}
  - &values {{ .values }}

# pulumi cloud username/organization
pulumiOrg: {{ .org }}

platform:
  - name: {{ .name }}
    domain: {{ .domain }}
    email: *email 
    region: *region
    repo: {{ .repo }}
    values: *values
  # aplVersion:
  # kubeVersion:  
  # nbTag:        
  # nodeCount:
  # nodeMax:      
  # nodeType:     
  # objPrefix:    
  # stack:  
  # tags: []
