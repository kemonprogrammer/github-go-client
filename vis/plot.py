import pandas as pd
import matplotlib.pyplot as plt
import json

# 1. Load and Clean
with open('custom-deploys.json', 'r') as f:
    data = json.load(f)

df = pd.DataFrame(data)

# Convert to datetime (handles missing keys automatically)
for col in ['created_at', 'updated_at', 'succeeded_at']:
    if col in df.columns:
        df[col] = pd.to_datetime(df[col], errors='coerce')

# 2. Setup Plot
plt.figure(figsize=(12, max(4, len(df) * 0.4)))
id_col = 'id' # or 'name'

# 3. Draw each deployment row
for i, row in df.iterrows():
    y_val = str(row[id_col])
    
    # Draw line from created_at to updated_at (if both exist)
    if pd.notnull(row.get('created_at')) and pd.notnull(row.get('updated_at')):
        plt.hlines(y=y_val, xmin=row['created_at'], xmax=row['updated_at'], 
                   color='skyblue', linewidth=3, zorder=1)
        
        # Add small dots at start/end of the line for clarity
        plt.plot([row['created_at']], [y_val], marker='o', color='gray', markersize=4)
        plt.plot([row['updated_at']], [y_val], marker='o', color='gray', markersize=4)

    # Add the Checkmark if succeeded_at is defined
    if pd.notnull(row.get('succeeded_at')):
        plt.plot([row['succeeded_at']], [y_val], 
                 marker='|', # docs: https://matplotlib.org/stable/api/markers_api.html
                 color='green', 
                 markersize=12, 
                 label='Succeeded' if i == 0 else "")

# 4. Formatting
plt.title('Deployment Durations & Success Markers', loc='left', fontsize=14)
plt.xlabel('Time')
plt.ylabel('Deployment')
plt.grid(True, axis='x', linestyle=':', alpha=0.5)

# Clean up duplicate legend entries
from matplotlib.lines import Line2D
custom_lines = [Line2D([0], [0], color='skyblue', lw=3),
                Line2D([0], [0], marker='|', color='w', markerfacecolor='green', markersize=10)]
plt.legend(custom_lines, ['Activity Period (Created -> Updated)', 'Success Marker'], 
           loc='upper right', bbox_to_anchor=(1, -0.1))

plt.tight_layout()
plt.show()

plt.savefig('deployment_timeline.png')
