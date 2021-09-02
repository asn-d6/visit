### Generates a bunch of fake json data for testing the visualization

import random

print('var items = ['),
start_epoch = random.randint(12341,512555)
for i in range(1000): # validators
    validator = random.randint(0,1243500)

    for epoch in range(start_epoch, start_epoch+10): # epochs
        distance = random.randint(0,64)
        print('{"validator_idx": %d, "epoch": %d, "distance": %d},' % (validator, epoch, distance))
print("]"),
