# Mandate cancellation

Owner: payments operations. Confirm the authenticated buyer request or signed
provider event, the mandate ID, and any active collection reservation.

Cancel through the mandate service, append the mandate event, and suspend
future collection before notifying the supplier. Existing obligations remain
open and require an alternative payment path. If cancellation is erroneous,
only a fresh provider authorization may reactivate the mandate; never edit the
event history.
