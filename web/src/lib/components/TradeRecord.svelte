<!--
  One example sale, played from agreement to settlement. It is labelled as an
  example and uses made-up businesses: it shows what a record looks like, not
  anything about real volumes.

  The resting style is the finished record. The animation only runs from an
  earlier state back to it, so with reduced motion, low data or no CSS
  animation support the reader simply sees the settled sale.
-->
<figure class="trade-record" aria-label="An example sale record">
	<header>
		<span>Example sale</span>
		<span class="ref">KR-0412</span>
	</header>

	<dl class="parties">
		<div>
			<dt>Seller</dt>
			<dd>A distributor in Onitsha</dd>
		</div>
		<div>
			<dt>Buyer</dt>
			<dd>A provisions shop in Aba</dd>
		</div>
		<div>
			<dt>Goods</dt>
			<dd>40 cartons of noodles</dd>
		</div>
		<div>
			<dt>Total</dt>
			<dd class="figure">₦480,000.00</dd>
		</div>
	</dl>

	<ol class="events">
		<li style="--at: 1.0s"><span>Terms accepted by both sides</span><time>12 Mar</time></li>
		<li style="--at: 1.6s"><span>Delivered, 40 of 40 cartons</span><time>14 Mar</time></li>
		<li style="--at: 2.3s"><span>Paid ₦160,000 · 1 of 3</span><time>28 Mar</time></li>
		<li style="--at: 3.0s"><span>Paid ₦160,000 · 2 of 3</span><time>11 Apr</time></li>
		<li style="--at: 3.7s"><span>Paid ₦160,000 · 3 of 3</span><time>25 Apr</time></li>
	</ol>

	<footer>
		<div class="balance">
			<span class="label">Left to pay</span>
			<!-- Four values share one cell and take turns; only the last one is
			     visible at rest. The live region is off because the animation is
			     decoration: the list above already says what happened. -->
			<span class="values figure" aria-hidden="true">
				<span class="b0">₦480,000.00</span>
				<span class="b1">₦320,000.00</span>
				<span class="b2">₦160,000.00</span>
				<span class="b3">₦0.00</span>
			</span>
			<span class="visually-hidden">₦0.00</span>
		</div>
		<span class="seal">Settled</span>
	</footer>
</figure>

<style>
	.trade-record {
		/* the stamp starts oversized; it must not widen the page while it lands */
		overflow: clip;
		margin: 0;
		border: 1px solid var(--color-border-strong);
		background: linear-gradient(176deg, #171b24, #101319 60%);
		box-shadow: var(--shadow-lg), var(--edge-lit);
		font-variant-numeric: tabular-nums;
		animation: record-in 800ms cubic-bezier(0.16, 0.7, 0.2, 1) 250ms both;
	}
	header {
		display: flex;
		justify-content: space-between;
		padding: 1rem 1.5rem;
		border-bottom: 1px solid var(--color-border);
		background: rgb(255 255 255 / 0.02);
		color: var(--color-muted);
		font-size: 0.78rem;
		font-weight: 550;
	}
	.ref {
		font-family: var(--font-mono);
		font-size: 0.74rem;
	}

	.parties {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 1rem 1.5rem;
		margin: 0;
		padding: 1.3rem 1.5rem;
	}
	.parties dt {
		color: var(--color-muted);
		font-size: 0.75rem;
	}
	.parties dd {
		margin: 0.2rem 0 0;
		font-size: 0.93rem;
		font-weight: 550;
	}
	.figure {
		font-weight: 600;
		letter-spacing: -0.01em;
	}

	/* The timeline: a rail on the left, a mark per event that lights as it
	   happens. */
	.events {
		position: relative;
		margin: 0;
		padding: 0.4rem 1.5rem 1.2rem 2.9rem;
		list-style: none;
		border-top: 1px solid var(--color-border);
	}
	.events::before {
		content: '';
		position: absolute;
		left: 1.83rem;
		top: 1.5rem;
		bottom: 2.1rem;
		width: 1px;
		background: var(--color-accent-ink);
		transform-origin: top;
		animation: rail 2.9s linear 1s both;
	}
	.events li {
		position: relative;
		display: flex;
		justify-content: space-between;
		gap: 1rem;
		padding: 0.72rem 0;
		font-size: 0.88rem;
		animation: event-on 450ms cubic-bezier(0.16, 0.7, 0.2, 1) var(--at) both;
	}
	.events li::before {
		content: '';
		position: absolute;
		left: -1.35rem;
		top: 1.02rem;
		width: 0.5rem;
		height: 0.5rem;
		background: var(--color-accent-ink);
		transform: rotate(45deg);
		animation: mark-on 450ms cubic-bezier(0.2, 0.9, 0.3, 1.2) var(--at) both;
	}
	time {
		flex: none;
		color: var(--color-muted);
		font-size: 0.8rem;
	}

	footer {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 1rem;
		padding: 1.1rem 1.5rem;
		border-top: 1px solid var(--color-border);
		background: rgb(0 0 0 / 0.25);
	}
	.balance {
		display: grid;
		gap: 0.2rem;
	}
	.label {
		color: var(--color-muted);
		font-size: 0.75rem;
	}
	.values {
		display: grid;
		font-size: 1.3rem;
	}
	.values > span {
		grid-area: 1 / 1;
		opacity: 0;
	}
	.values .b3 {
		opacity: 1;
		animation: appear 300ms ease 3.7s both;
	}
	/* each earlier balance is only visible while its step is current */
	.values .b0 {
		animation: hold 2.3s steps(1, end) 0s;
	}
	.values .b1 {
		animation: hold 0.7s steps(1, end) 2.3s;
	}
	.values .b2 {
		animation: hold 0.7s steps(1, end) 3s;
	}

	/* The seal's last appearance in the system: a sale that settled. */
	.seal {
		padding: 0.35rem 0.75rem;
		color: var(--color-accent-ink);
		border: 1.5px solid var(--color-accent-ink);
		font-size: 0.66rem;
		font-weight: 700;
		letter-spacing: 0.18em;
		text-transform: uppercase;
		transform: rotate(-4deg);
		animation: stamp 520ms cubic-bezier(0.3, 1.4, 0.5, 1) 4.2s both;
	}

	.visually-hidden {
		position: absolute;
		width: 1px;
		height: 1px;
		overflow: hidden;
		clip-path: inset(50%);
		white-space: nowrap;
	}

	@keyframes record-in {
		from {
			opacity: 0;
			transform: translateY(24px);
		}
	}
	@keyframes rail {
		from {
			transform: scaleY(0);
		}
	}
	@keyframes event-on {
		from {
			opacity: 0.28;
		}
	}
	@keyframes mark-on {
		from {
			background: transparent;
			outline: 1px solid var(--color-border-strong);
			transform: rotate(45deg) scale(0.6);
		}
	}
	@keyframes hold {
		0%,
		100% {
			opacity: 1;
		}
	}
	@keyframes appear {
		from {
			opacity: 0;
		}
	}
	@keyframes stamp {
		from {
			opacity: 0;
			transform: rotate(-12deg) scale(1.9);
		}
	}

	@media (max-width: 480px) {
		.parties {
			grid-template-columns: 1fr;
		}
		header,
		.parties,
		footer {
			padding-inline: 1.15rem;
		}
		.events {
			padding-right: 1.15rem;
		}
	}
</style>
