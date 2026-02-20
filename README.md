> **Note:** This is a fork of the [Dropbox LLAMA](https://github.com/dropbox/llama) project.
> It has been modified to export Prometheus metrics instead of InfluxDB and includes
> Docker deployment support.


# UDProbe
[Read the Docs](https://udprobe.readthedocs.io/en/latest/)

UDProbe (mix of UDP and Probe) is a library for testing and measuring network loss and latency between distributed endpoints.

It does this by sending UDP datagrams/probes from **collectors** to **reflectors** and measuring how long it takes for them to return, if they return at all. UDP is used to provide ECMP hashing over multiple paths (a win over ICMP) without the need for setup/teardown and per-packet granularity (a win over TCP).

## Why Is This Useful

[Black box testing](https://en.wikipedia.org/wiki/Black-box_testing) is critical to the successful monitoring and operation of a network. While collection of metrics from network devices can provide greater detail regarding known issues, they don't always provide a complete picture and can provide an overwhelming number of metrics. Black box testing with UDProbe doesn't care how the network is structured, only if it's working. This data can be used for building KPIs, observing big-picture issues, and guiding investigations into issues with unknown causes by quantifying which flows are/aren't working. See this article on [probers](https://medium.com/cloudprober/why-you-need-probers-f38400f5830e) for a good explanation on how this works.

Network operators often find this useful for gauging the impact of network issues on internal traffic, identifying the scope of impact, and locating issues for which they had no other metrics (internal hardware failures, circuit degradations, etc).

**Even if you operate entirely in the cloud** UDProbe can help identify reachability and network health issues between and within regions/zones.

## Why UDProbe?
- **Lightweight**: Docker images are ~30MB in size. Directly built binaries are around ~10MB in size.
- **Simple Configuration**: Collectors require minimal configuration and reflectors require no configuration (ideal for remote site deployments).
- **UDP Based Probing**: Cycling UDP source ports allows for better coverage over ECMP paths.
- **QOS Support**: Probes can be sent with different TOS values to monitor different traffic classes.
- **Extensible**: Written in Go, easily modifiable to support different environments.
- **Prometheus Support**: Out of the box integration with Prometheus allowing for straightforward alerting and dashboarding.

## Get Started
Jump in with the [Getting Started](https://udprobe.readthedocs.io/en/latest/#quick-start) guide and get up in running with a few minutes.

## Architecture
Learn more about the system architecture over on our [Architecture](https://udprobe.readthedocs.io/en/latest/architecture/) page.
## Ongoing Development

This is a fork of the original Dropbox LLAMA project. The original was built during a [Dropbox Hack Week](https://www.theverge.com/2014/7/24/5930927/why-dropbox-gives-its-employees-a-week-to-do-whatever-they-want). This fork is currently in early development with significant changes including migration to Prometheus metrics and modernized dependencies. The API and config format may continue to evolve.

## Contributing

This is a very early stage project. Contributions are welcome, but please check with the maintainer first before submitting pull requests. We appreciate your interest in improving UDProbe!

## Acknowledgements/References

* Inspired by: <https://www.youtube.com/watch?v=N0lZrJVdI9A>
    * With slides: <https://www.nanog.org/sites/default/files/Lapukhov_Move_Fast_Unbreak.pdf>
* Concepts borrowed from: <https://github.com/facebook/UdpPinger/>
* Looking for the legacy Python version of llama?: https://github.com/dropbox/llama-archive
